##MongoDB / PostgreSQL JSONB Benchmarking Tool


MPJBT was wrote to gather data for our Percona Live EU 2017 talk - you can see our slides [here](https://docs.google.com/presentation/d/1c2RihL5G3teT0sxcngzjIs8ni6aSeAcL8Nzmvy3nj8Y/edit?usp=sharing). 

We found some *interesting* things.



## What's here?
* `mpjbt` - benchmark tool source code (see releases for pre-built binaries)
* `scripts/run-tests.sh` - bash script to run all workloads and push to a git remote
* `scripts/run-suite.sh` - bash script to run `run-tests.sh` with a variety of different parameters
* `scripts/vfs.d` - dtrace script to measure VFS call latency (`./vfs.d <execname>`, for example `./vfs.d mongod`)

The bash scripts were developed to let us run long unattended tests over night - you'll probably have to tweak them for your use.

## Features
* Pluggable drivers - supports PostgreSQL and MongoDB now, but should be easy to add others
* Randomised records
* Pads records out to test larger documents (1kb, 1mb, etc)
* Supports high numbers of concurrent workers
	* Care has been taken to avoid locks/contention outside of the drivers
* Two different random distributions supported
	* Uniform - good at cache busting
	* Zipfian - good at hitting the cache
* Support for configuring the behaviour of the underlying driver
	* Request different write concerns, set timeouts, etc
* Builds a histogram for request durations - don't just use the average throughput!
	* Breaks results down for each operation
	* Dump histogram data as a CSV 

## Workloads
* **insert**: insert records with a monotonically increasing ID
* **insert-update**: same as "insert' but immediately updates the record
* **insert-select**: same as insert, but immediately reads the record
* **insert5-select95**: insert a record 5% of the time, and read the most recent record the other 95%
* **select-uniform**: read a totally random record
* **select-zipfian**: read a random record heavily weighted towards the most recent
* **select-update-uniform**: same as select-uniform, but the record is immediately updated
* **select-update-zipfian**: same as select-zipfian, but the record is immediately updated
* **update-zipfian**: update a record, weighted towards the highest IDs
* **update-uniform**: update a random record
* **read-range**: perform a range query on the age field (`age > 45 AND age < 75`)

### Notes
* We've seen a significant speed improvement using `binary_parameters=yes` when connecting to Postgres
* Take into account the drivers used ([pq](https://github.com/lib/pq) and GlobalSign's fork of [mgo](https://github.com/globalsign/mgo)) will have differing performance
* This was built fairly quickly so we could grab data - be kind!
* Feel free to open PR's - I'll actively maintain this after the conference if it's useful to someone
* If you do open a PR, please maintain the lockless behaviour (outside of the drivers obviously)
* Multiple operation workloads are run sequentially per worker - this could be changed in the future (or just run two copies)
* I really hate releasing something without unit tests - I just ran out of time!
* Also I admit, we're not the best at coming up with names