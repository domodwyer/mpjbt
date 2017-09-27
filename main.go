package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/c2h5oh/datasize"
	"github.com/domodwyer/mpjbt/mongo"
	"github.com/domodwyer/mpjbt/plan"
	"github.com/domodwyer/mpjbt/postgres"
)

var (
	endpoint, tableName, histPath, paddingSize string
	updateFreq, readFreq, numWorkers           uint64
	opsMax                                     uint64

	workload string

	versionTag  = "unknown"
	versionDate = "unknown"
)

func init() {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.StringVar(&endpoint, "connect", "", "Connection string")
	fs.StringVar(&histPath, "histogram", "", "Histogram output file path (CSV)")
	fs.StringVar(&tableName, "table", "test", "Table/collection name")
	fs.StringVar(&paddingSize, "padding", "0", "Amount of binary padding in the records (valid suffixes: kb, mb)")
	fs.StringVar(&workload, "workload", "insert", "Workload name")
	fs.Uint64Var(&opsMax, "ops", 0, "Number of `operations` to perform (0 == unlimited)")
	fs.Uint64Var(&numWorkers, "workers", 30, "Number of concurrent workers")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Build %s (%s)\n\n", versionTag, versionDate)
		fs.PrintDefaults()

		var info = `
Available workloads:
	insert:
		Insert records with a monotonically increasing ID
	insert-update:
		Same as "insert' but immediately updates the record
	insert-select:
		Same as insert, but immediately reads the record
	insert5-select95:
		Insert a record 5% of the time, and read the most recent record the other 95%
	select-uniform:
		Read a totally random record
	select-zipfian:
		Read a random record heavily weighted towards the most recent
	select-update-uniform:
		Same as select-uniform, but the record is immediately updated
	select-update-zipfian:
		Same as select-zipfian, but the record is immediately updated
	update-zipfian:
		Update a record, weighted towards the highest IDs
	update-uniform:
		Update a random record
	read-range:
		Perform a range query on the age field (age > 45 AND age < 75)

Postgres dial string parameters:
	See https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters

	Example: "postgres://localhost/test?sslmode=disable;binary_parameters=yes"

Mongo dial string parameters:
	In addition to the driver dial parameters (https://godoc.org/github.com/globalsign/mgo#Dial)
	there are several more flags for specifying session parameters:
	
	readConcern: (docs: https://docs.mongodb.com/manual/reference/read-concern/)
		majority / local / linearizable
	writeConcern: (docs: https://docs.mongodb.com/manual/reference/write-concern/)
		majority / <number>
	journal: (docs: https://docs.mongodb.com/manual/core/journaling/)
		true / false
	fsync: (docs: https://docs.mongodb.com/manual/reference/command/fsync/)
		true / false
	
	The default values of the above are the defaults specified by MonogDB.

	Example: "mongodb://localhost/test?journal=true&writeConcern=majority"
`

		fmt.Fprintf(os.Stderr, "\n%s\n", info)
	}
	fs.Parse(os.Args[1:])
	if endpoint == "" {
		fs.Usage()
		os.Exit(1)
	}
}

func main() {
	var histW = ioutil.Discard
	if histPath != "" {
		f, err := os.Create(histPath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		histW = f
	}

	// Parse the record padding
	var padding datasize.ByteSize
	if err := padding.UnmarshalText([]byte(paddingSize)); err != nil {
		log.Fatalf("padding: %v", err)
	}

	// Get the correct provider for this DB type
	db, err := getDB(endpoint, tableName)
	if err != nil {
		log.Fatal(err)
	}

	// Create the work plan
	dbplan := plan.New(opsMax, padding.Bytes())
	if err := setWorkload(workload, dbplan, db); err != nil {
		log.Fatal(err)
	}

	// Clean-up handler
	var sigInfo = make(chan os.Signal, 1)
	signal.Notify(sigInfo, syscall.SIGINT)

	// If the user hits Ctrl+C, stop the plan and aggregate the statistics.
	go func() {
		<-sigInfo
		fmt.Printf("\nStopping... Ctrl+C again to force\n")
		dbplan.Stop()
		<-sigInfo
		fmt.Printf("\nForcing...")
		os.Exit(1)
	}()

	// Go!
	results := dbplan.Run(numWorkers, os.Stdout)

	// Output the latency histograms as CSV files to histW.
	reportHistograms(histW, results)
}

// getDB parses endpoint and returns a database provider based on the scheme.
//
// Available endpoint scehmes are "mongodb" and "postgres".
func getDB(endpoint, tableName string) (dbProvider, error) {
	purl, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	var provider dbProvider

	switch purl.Scheme {
	case "mongodb":
		provider, err = mongo.NewProvider(purl, tableName)
		if err != nil {
			return nil, err
		}

	case "postgres":
		provider, err = postgres.NewProvider(purl, tableName)
		if err != nil {
			return nil, err
		}

	default:
		log.Fatalf("unknown scheme '%s', valid: mongodb postgres", purl.Scheme)
	}

	return provider, nil
}

// reportHistograms writes runtime configuration and results to w as a CSV file.
func reportHistograms(w io.Writer, results []plan.Result) {
	// Print some run statistics
	cw := csv.NewWriter(w)
	cw.WriteAll([][]string{
		{"Endpoint:", endpoint},
		{"Table:", tableName},
		{"RecordLimit:", strconv.FormatUint(opsMax, 10)},
		{"Workers:", strconv.FormatUint(numWorkers, 10)},
		{"PaddingSize:", paddingSize},
		{"Workload:", workload},
		{},
	})
	defer cw.Flush()

	for _, op := range results {
		if op.Histogram.Count == 0 {
			return
		}

		// Print to stdout
		fmt.Printf("\n%s latency:\n", op.Name)
		op.Histogram.Print(os.Stdout)

		// Write to w
		fmt.Fprintf(w, "\n%s latency\n", op.Name)
		if err := op.Histogram.WriteCSV(w); err != nil {
			log.Printf("error writing histogram: %v", err)
		}
	}
}
