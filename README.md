# Finance Forecast

## Purpose

Finance Forecast is meant to be a simple tool to read in a config file that outlines various financial events (i.e. anything related to income or spending, including loans) and associated dates and then run simulations on one or more scenarios to see how these turn out. The more specific the events are the more accurate the simulation should be. Output defaults to printing to the console on STDOUT in a pretty format, but CSV format for easy import to other software may also be specified.

The general goal is to help provide users with a reasonable best-guess guide for financial decisions. By expressing different choices in different scenarios and then running the simulation the user can see how things will generally play out over the long-term.

## Assumptions

### Simulation

* The simulation starts processing events the next calendar month after when this is run; in other words the starting value should be taking into account the current month.

### Loan Handling

Loans are currently assumed to be compounded monthly. It is a future work item to allow specifying the compounding period.

The simulation makes some assumptions regarding escrow handling for simplicity's sake; if your needs require more precise handling then you could handle escrow using events and not define it (or set it to 0) in the loan.

* If escrow is defined the simulation assumes that the accumulated escrow for that year will be refunded if:
  1. The loan is paid off early (but not on December)
  1. The loan matures (but not on December)
* If escrow is defined and the asset is _not_ sold off then the escrow will be extrapolated to an annual expense paid in December

## Usage

Run `finance-forecast --help` to print command-line flags which include specifying the YAML-formatted config file's location and the output format.

Sample usage:

```
finance-forecast -config=./config.yaml.example -output-format=csv
```

Output:

```
"date","amount (current path)","notes (current path)","amount (new home purchase)","notes (new home purchase)"
"2020-07","30000.00","","30000.00",""
"2020-08","29670.24","","29670.24",""
"2020-09","29340.47","","29340.47",""
"2020-10","29000.71","","30111.82",""
...
"2031-04","22984.27","","129321.71",""
"2031-05","23085.46","","8730.89","paying off asset 5678 Street Address for 125301.49"
"2031-06","23186.66","","11041.56",""
"2031-07","23277.86","","13342.23",""
"2031-08","23379.06","","15652.90",""
...
"2089-09","285469.91","","566730.97",""
"2089-10","286424.91","","567685.97",""
"2089-11","287389.91","","568650.97",""
"2089-12","282354.91","","562415.97",""
"2090-01","283309.91","","563370.97",""
```

## Future Work

* Implement tests
* Support periodic extra principal payments on loans
* Ability to declare categories for events and tabulate average spending and income based on category
* Consider methods for simple optimization by configuring supported values as within a range and optimize a cost function
* Handle inflation
* Allow specifying compounding period for loans
* Handle investment scenarios with reallocating asset distribution over time
* Produce charts of the different scenarios without having to import the CSV output in spreadsheet software
* A GUI might be interesting?

