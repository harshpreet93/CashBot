package main

import "fmt"
import (
	"github.com/preichenberger/go-gdax"
	"time"
	"errors"
	"math"
)

const MAX_CONCURRENT_OPEN_POSITIONS = 3
const MAX_OPEN_POSITIONS_PER_CURRENCY  = 1
const MAX_USD_BUY_COST = 25
const DESIRED_PROFIT_PERCENT_PER_FLIP = 2.5

func CreateOrder(client *gdax.Client, price float64, amount float64, currencyCode string, isBuy bool) (gdax.Order, error) {
	side := "sell"
	if isBuy {
		side = "buy"
	}

	order := gdax.Order{
		Price: price,
		Size: amount,
		Side: side,
		ProductId: currencyCode+"-USD",
	}
	//time, err := client.GetTime()
	//if(err != nil) {
	//	fmt.Println("unable to get servertime")
	//	return order, err
	//}
	//cancelAfter := strconv.FormatFloat(time.Epoch+120.0, 'f', 6, 64)
	if isBuy {
		order = gdax.Order{
			Price: price,
			Size: amount,
			Side: side,
			ProductId: currencyCode+"-USD",
			TimeInForce: "GTT",
			CancelAfter: "min",
		}
	}

	savedOrder, err := client.CreateOrder(&order)

	if err != nil {
		println(err.Error())
	}

	return savedOrder, nil
}

func getOptimalBuyPrice(client *gdax.Client, currencyCode string) (float64, error) {
	stats, err :=client.GetStats(currencyCode+"-USD")

	if err != nil {
		return 0.0, err
	}

	return stats.Last, nilh
}

func getSellPrice(buyPrice float64) float64 {
	return buyPrice + (buyPrice * 0.01)
}

func isThereOpenOrderFor(client *gdax.Client, currencyCode string) (bool, error) {
	listOrderParams := gdax.ListOrdersParams{
		Status: "open",
		Pagination: gdax.PaginationParams{
			Limit: 100,
			Before: "",
			After: "",
		},
	}
	cursor := client.ListOrders(listOrderParams)
	var orders []gdax.Order
	for cursor.HasMore {
		err := cursor.NextPage(&orders)
		if err != nil {
			fmt.Errorf("problem getting orders", err)
			return false, err
		}

		for _, order := range orders {
			if order.Status == "open" && order.ProductId == currencyCode+"-USD"{
				return true, nil
			}
		}

	}
	return false, nil
}

func iHaveSome(client *gdax.Client, currencyCode string) (bool, error) {
	accounts, err := client.GetAccounts()

	if err != nil {
		return false, err
	}

	for _, account := range accounts {
		if account.Currency == currencyCode && account.Balance > 0.0001 {
			return true, nil
		}
	}
	return false, nil

}

func getLastFillFor(client *gdax.Client, currencyCode string, side string) (*gdax.Fill, error) {
	allFills := client.ListFills()
	var fillPage []gdax.Fill
	for allFills.HasMore {
		err := allFills.NextPage(&fillPage)

		if err != nil {
			return nil, err
		}
		for _, fill := range fillPage {
			if fill.ProductId == currencyCode+"-USD" && fill.Side == side && fill.Settled {
				return &fill, nil
			}
		}
	}
	return nil, errors.New("error getting last fill for "+currencyCode)
}

func Round(f float64) float64 {
	return math.Floor(f + .5)
}

func RoundPlus(f float64, places int) (float64) {
	shift := math.Pow(10, float64(places))
	return Round(f * shift) / shift
}

func sell(client *gdax.Client, currencyCode string) {
	lastFill, err :=  getLastFillFor(client, currencyCode, "buy")

	if err != nil {
		return
	}

	fee := lastFill.Fee
	price := lastFill.Price

	sellPrice := (fee + price) + ((fee + price)*0.01)

	fmt.Printf("selling %s inventory at %f bought at %f size is %f \n", currencyCode, sellPrice, lastFill.Price, lastFill.Size)

	CreateOrder(client, RoundPlus( sellPrice, 2), lastFill.Size, currencyCode, false)

}

func buy(client *gdax.Client, currencyCode string, usdAmount float64) {
	optimalBuyPrice, err := getOptimalBuyPrice(client, currencyCode)

	if err != nil {
		fmt.Errorf("cannot buy due to ", err)
		return
	}
	size := usdAmount / optimalBuyPrice
	CreateOrder(client, optimalBuyPrice, RoundPlus(size, 5), currencyCode, true)
}

func startFlipLoop(client *gdax.Client, currencyCode string, usdAmount float64)  {
	for {
		doesOrderAlreadyExist, err := isThereOpenOrderFor(client, currencyCode)

		if doesOrderAlreadyExist || err != nil {
			fmt.Printf("order already exists, trying again after 1 minute \n")
			time.Sleep(1 * time.Minute)
			continue
		}

		iAlreadyHaveSome, err := iHaveSome(client, currencyCode)

		if err != nil || iAlreadyHaveSome {
			sell(client, currencyCode)
			fmt.Println("selling\n")
			continue
		}

		buy(client, currencyCode, usdAmount)
		fmt.Println("waiting after buying\n")

		}
}

func main() {
	// or unsafe hardcode way
	key := "XXXXXXXXXXXXXXXXXXXXXX"
	secret := "XXXXXXXXXXXXXXXXXXXXXX"
	passphrase := "XXXXXXXXXXXXXXXXXX"

	client := gdax.NewClient(secret, key, passphrase)

	startFlipLoop(client, "BTC", 100.0)
}
