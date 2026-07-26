# currency-quotes-service
Go service for currency quotes: request an update, fetch the result by ID. Background worker, PostgreSQL, HTTP/JSON API.

Supported currencies: CAD, EUR, GBP, MXN, USD.

The service provides quotes with a precision of 6 decimal places.

## Rate provider

Rates are sourced from Banco de México (Banxico) reference rates via [Frankfurter](https://frankfurter.dev) (`providers=BANXICO`). Central bank reference rates are published once per business day, so rates only actually change on that cadence regardless of how often an update is requested. `RatesProvider` is defined as an interface, so a different provider can be swapped in if a task needs more frequent updates or coverage from another source.






