package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
)

const (
	gcpCred           = "cowCounterCredentials.json"
	defaultChanBuffer = 1e2
)

var verboseFlag *bool
var takePictures *bool
var savePcap *bool
var bqInserter *bigquery.Inserter
var db *sql.DB

func main() {
	// command line arguments
	createNewDb := flag.Bool("createdb", false, "Use this flag if a new database should be created")
	verboseFlag = flag.Bool("v", true, "Print sort events and stays to the command line")
	takePictures = flag.Bool("p", false, "Take pictures of the sortings using the ip cameras")
	savePcap = flag.Bool("w", false, "Wheter to save a pcap file containing the unfiltered raw packets")
	dbName := flag.String("db", "sortings01.db", "Path to the database")
	flag.Parse()

	// Get an inserter for the BigQuery table
	bqInserter = bqInit()

	// database stuff
	if *createNewDb {
		createDb(*dbName)
	}
	db = openDb(*dbName)
	defer db.Close()

	// create channels
	srChan := make(chan SortEvent, defaultChanBuffer)
	seToStaysChan := make(chan SortEvent, defaultChanBuffer)
	staysToSaveChan := make(chan Stay, defaultChanBuffer)

	// start goroutine for saving and displaying sorting events
	go SaveAndShowSE(srChan, seToStaysChan)

	// analyze the SortEvents and convert them into stays
	go SortingResultsToStays(seToStaysChan, staysToSaveChan)

	// save and display stays
	go handleStays(staysToSaveChan)

	// start capturing packets and start goroutine to handle packets
	// pcapIn, err := pcap.OpenLive("eth0", 400, true, pcap.BlockForever)
	pcapIn, err := pcap.OpenOffline("data/20220320_RoboCap03.cap")
	check(err)
	packetSource := gopacket.NewPacketSource(pcapIn, pcapIn.LinkType())
	go handlePacket(packetSource.Packets(), srChan)

	// just do nothing, everything that happens now happens in non-main goroutines
	for {
		time.Sleep(100 * time.Second)
	}
}

func SaveAndShowSE(seIn <-chan SortEvent, seToStaysChan chan<- SortEvent) {
	gcpStorageClient := initTakePicture()

	for {
		se := <-seIn
		var objName string
		if *takePictures {
			objName, err := takePicture(se, gcpStorageClient)
			if err != nil {
				fmt.Println("ERROR:", err)
			}
			if objName == "" {
				fmt.Println("Empty object name")
			} else {
				fmt.Println(objName)
			}
		}
		bqInsertSE(bqInserter, se, objName)
		insertSortEvent(se, objName, db)
		if *verboseFlag {
			ShowSortEvent(se)
		}
		seToStaysChan <- se
	}
}

func handleStays(stIn <-chan Stay) {
	for {
		st := <-stIn

		insertStay(st, db)
		if *verboseFlag {
			ShowStay(st)
		}
	}
}

func handlePacket(packetsChan <-chan gopacket.Packet, srChan chan<- SortEvent) {
	var pcapw *pcapgo.Writer
	// init pcap file
	if *savePcap {
		filename := fmt.Sprintf("raw_%v.pcap", time.Now().Format("2006-01-02_15-04-05"))
		f, err := os.Create(filename)
		check(err)
		defer f.Close()

		pcapw = pcapgo.NewWriter(f)
		if err := pcapw.WriteFileHeader(1600, layers.LinkTypeEthernet); err != nil {
			log.Fatalf("WriteFileHeader: %v", err)
		}
	}

	for {
		packet := <-packetsChan

		// save packet
		if *savePcap {
			err := pcapw.WritePacket(packet.Metadata().CaptureInfo, packet.Data())
			if err != nil {
				fmt.Println("ERROR:", err)
			}
		}

		if udp := packet.Layer(layers.LayerTypeUDP); udp != nil && len(udp.LayerPayload()) > 4 {
			if udp.LayerPayload()[0] == 0x00 && udp.LayerPayload()[1] == 0x05 && udp.LayerPayload()[2] == 0x01 && udp.LayerPayload()[3] == 0x0a {
				if len(udp.LayerPayload()) == 18 && packet.Metadata().CaptureLength == 60 {
					continue
				}
				if len(udp.LayerPayload()) == 222 && packet.Metadata().CaptureLength == 264 {
					se := decodeSortEvent(packet)
					srChan <- se

					continue
				}
			}
		}
	}
}

func printHex(data []byte) {
	for _, b := range data {
		if b == 0x0 {
			fmt.Printf("\033[38;5;%dm", 240)
			fmt.Printf("%02x ", b)
			fmt.Printf("\033[0m")
		} else {
			fmt.Printf("%02x ", b)
		}
	}
}

func ShowPacketInfo(packet gopacket.Packet) {
	timestamp := packet.Metadata().Timestamp
	readable_timestamp := timestamp.Format("2006-01-02 15:04:05")
	dst := packet.NetworkLayer().NetworkFlow().Dst()
	src := packet.NetworkLayer().NetworkFlow().Src()
	payload := packet.Layer(layers.LayerTypeUDP).LayerPayload()

	fmt.Printf("\n\n")
	fmt.Println(readable_timestamp, src, "->", dst)
	fmt.Printf("PAYLOAD: % x\n", payload)
	printHex(payload)
	fmt.Printf("\n")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
