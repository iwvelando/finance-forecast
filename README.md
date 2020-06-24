# Finance Forecast

## Purpose

Finance Forecast is meant to be a simple tool to read in a config file that outlines various financial events (i.e. anything related to income or spending) and associated dates and then run simulations on one or more scenarios to see how these turn out. The more specific the events are the more accurate the simulation should be. Output defaults to printing to the console on STDOUT in a pretty format, but CSV format for easy import to other software may also be specified.

## Usage

Run `finance-forecast --help` to print command-line flags which include specifying the YAML-formatted config file's location and the output format.

Sample usage:

```
finance-forecast -config=./config.yaml.example -output-format=csv
```

Output:

```
"date","amount (current path)","amount (alternative path)"
"2020-06","30000.00","30000.00"
"2020-07","31189.56","31189.56"
"2020-08","32389.12","32389.12"
"2020-09","33588.68","33588.68"
"2020-10","34778.24","35889.35"
"2020-11","35977.80","38200.02"
"2020-12","45068.59","40510.69"
"2021-01","46258.15","42811.36"
"2021-02","47457.71","45122.03"
...
"2089-04","916490.03","1299709.52"
"2089-05","917455.03","1300674.52"
"2089-06","918420.03","1301639.52"
"2089-07","919375.03","1302594.52"
"2089-08","920340.03","1303559.52"
"2089-09","921305.03","1304524.52"
"2089-10","922260.03","1305479.52"
"2089-11","923225.03","1306444.52"
"2089-12","924190.03","1307409.52"
"2090-01","925145.03","1308364.52"
```

## Future Work

* Implement tests
* I would like to integrate loan amortization schedules and balance thresholds prior to early pay-offs to make simulating those scenarios easier
* I would like to be able to declare categories for events and tabulate average spending and income based on category
* I would like this to be able to produce charts of the different scenarios without having to import the CSV output in spreadsheet software
* A GUI might be interesting?

