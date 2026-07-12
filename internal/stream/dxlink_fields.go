package stream

// dxEventTypes are the DxLink event types we subscribe to for every symbol.
// Verified live via cmd/apiprobe against tastytrade DxLink.
var dxEventTypes = []string{"Quote", "Trade", "Summary", "Profile", "Greeks"}

// dxFieldSpec mirrors acceptEventFields sent in FEED_SETUP. Order is the wire
// order of COMPACT FEED_DATA values (from live probe / dxFeed schemas).
var dxFieldSpec = FieldSpec{
	"Quote": {
		"eventSymbol", "eventTime", "sequence", "timeNanoPart",
		"bidTime", "bidExchangeCode", "bidPrice", "bidSize",
		"askTime", "askExchangeCode", "askPrice", "askSize",
	},
	"Trade": {
		"eventSymbol", "eventTime", "timeNanoPart", "exchangeCode",
		"price", "change", "size", "dayVolume", "dayTurnover",
		"tickDirection", "extendedTradingHours",
	},
	"Summary": {
		"eventSymbol", "dayId", "dayOpenPrice", "dayHighPrice", "dayLowPrice",
		"dayClosePrice", "dayClosePriceType", "prevDayId", "prevDayClosePrice",
		"prevDayClosePriceType", "prevDayVolume", "openInterest",
	},
	"Profile": {
		"eventSymbol", "description", "shortSaleRestriction", "tradingStatus",
		"statusReason", "haltStartTime", "haltEndTime", "highLimitPrice", "lowLimitPrice",
		"high52WeekPrice", "low52WeekPrice", "beta", "earningsPerShare",
		"dividendFrequency", "exDividendAmount", "exDividendDayId", "shares", "freeFloat",
	},
	"Greeks": {
		"eventSymbol", "eventTime", "eventFlags", "index", "time", "sequence",
		"price", "volatility", "delta", "gamma", "theta", "rho", "vega",
	},
}
