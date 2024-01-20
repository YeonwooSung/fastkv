package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/YeonwooSung/fastkv/src/bloomfilter"
)

// *** App struct and methods ***

type App struct {
	db          *leveldb.DB
	mlock       sync.Mutex
	lock        map[string]struct{}
	bloomfilter *bloomfilter.ScalableBloomFilter

	// params
	uploadids  map[string]bool
	volumes    []string
	fallback   string
	replicas   int
	subvolumes int
	protect    bool
	md5sum     bool
	voltimeout time.Duration
}

func (a *App) UnlockKey(key []byte) {
	a.mlock.Lock()
	delete(a.lock, string(key))
	a.mlock.Unlock()
}

func (a *App) LockKey(key []byte) bool {
	a.mlock.Lock()
	defer a.mlock.Unlock()
	if _, prs := a.lock[string(key)]; prs {
		return false
	}
	a.lock[string(key)] = struct{}{}
	return true
}

func (a *App) GetRecord(key []byte) Record {
	// use bloom filter to minimize disk access
	has_key, err := a.bloomfilter.Test(key)
	if err != nil {
		log.Printf("Bloom filter test failed: %s", err)
		return Record{[]string{}, HARD, ""}
	}
	if !has_key {
		return Record{[]string{}, HARD, ""}
	}

	// get the record from leveldb
	data, err := a.db.Get(key, nil)
	rec := Record{[]string{}, HARD, ""}
	if err != leveldb.ErrNotFound {
		rec = toRecord(data)
	}
	return rec
}

/**
 * PutRecord adds a record to the database.
 *
 * @param key The key to add.
 * @param rec The record to add.
 * @return True if the record was added, false otherwise.
 */
func (a *App) PutRecord(key []byte, rec Record) bool {
	// add key to bloom filter first
	if err := a.bloomfilter.Add(key); err != nil {
		log.Printf("Bloom filter add failed: %s", err)
		return false
	}

	put_err := a.db.Put(key, fromRecord(rec), nil)
	return put_err == nil
}

// *** Entry Point ***

func main() {
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100
	rand.New(rand.NewSource(time.Now().Unix()))

	port := flag.Int("port", 9000, "Port for the server to listen on")
	pdb := flag.String("db", "", "Path to leveldb")
	fallback := flag.String("fallback", "", "Fallback server for missing keys")
	replicas := flag.Int("replicas", 3, "Amount of replicas to make of the data")
	subvolumes := flag.Int("subvolumes", 10, "Amount of subvolumes, disks per machine")
	pvolumes := flag.String("volumes", "", "Volumes to use for storage, comma separated")
	protect := flag.Bool("protect", false, "Force UNLINK before DELETE")
	verbose := flag.Bool("v", false, "Verbose output")
	md5sum := flag.Bool("md5sum", true, "Calculate and store MD5 checksum of values")
	voltimeout := flag.Duration("voltimeout", 1*time.Second, "Volume servers must respond to GET/HEAD requests in this amount of time or they are considered down, as duration")
	flag.Parse()

	volumes := strings.Split(*pvolumes, ",")
	command := flag.Arg(0)

	if command != "server" && command != "rebuild" && command != "rebalance" {
		fmt.Println("Usage: ./fastkv <server, rebuild, rebalance>")
		flag.PrintDefaults()
		return
	}

	if !*verbose {
		log.SetOutput(io.Discard)
	} else {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	}

	if *pdb == "" {
		panic("Need a path to the database")
	}

	if len(volumes) < *replicas {
		panic("Need at least as many volumes as replicas")
	}

	db, err := leveldb.OpenFile(*pdb, nil)
	if err != nil {
		panic(fmt.Sprintf("LevelDB open failed: %s", err))
	}
	defer db.Close()

	fmt.Printf("volume servers: %s\n", volumes)
	app := App{db: db,
		lock:       make(map[string]struct{}),
		uploadids:  make(map[string]bool),
		volumes:    volumes,
		fallback:   *fallback,
		replicas:   *replicas,
		subvolumes: *subvolumes,
		protect:    *protect,
		md5sum:     *md5sum,
		voltimeout: *voltimeout,
	}

	if command == "server" {
		http.ListenAndServe(fmt.Sprintf(":%d", *port), &app)
	} else if command == "rebuild" {
		app.Rebuild()
	} else if command == "rebalance" {
		app.Rebalance()
	}
}
