package main

import (
	"net/http"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os"
	"time"
	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Block struct {
	Index 		int  	// location where records stored
	Timestamp 	string  // record time
	BPM  		int    	// data
	Hash  		string	// SHA256 for the record
	PreHash		string	// SHA256 for the pre record of current record
}

type Message struct {
	BPM  		int
}
var Blockchain []Block;
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		genesisBlock := Block{0,t.String(), 0, "", ""}
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain,genesisBlock)
	}()

	log.Fatal(run())
}

func calculateHash(block Block)string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PreHash
	h := sha256.New()
	h.Write([]byte(record));
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func generateBlock(preBlock Block, BPM int)(Block,error) {
	var newBlock Block
	t := time.Now()

	newBlock.Index = preBlock.Index+1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PreHash = preBlock.Hash
	newBlock.Hash = calculateHash(newBlock)
	return newBlock, nil
}

func isBlockValid(newBlock Block, preBlock Block) bool {
	if (preBlock.Index + 1 != newBlock.Index){
		return false
	}
	if (preBlock.Hash != newBlock.PreHash) {
		return false
	}
	if (calculateHash(newBlock) != newBlock.Hash) {
		return false
	}

	return true
}

// if what we accepted the length of blockchains is longer than the chain we hold, 
// so we update our chain as what we accpeted.
func replaceChain(newBlocks []Block) {
	if (len(newBlocks) > len(Blockchain)) {
		Blockchain = newBlocks
	}
}

func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on ", os.Getenv("ADDR"))
	s := &http.Server {
		Addr : 			":"+httpAddr,
		Handler: 		mux,
		ReadTimeout: 	10 * time.Second,
		WriteTimeout: 	10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if err := s.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
	return muxRouter
}

func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain,""," ")
	if nil != err {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w,string(bytes))
}

func handleWriteBlock(w http.ResponseWriter, r * http.Request) {
	var messsage Message

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&messsage); err != nil {
		respondWithJSON(w,r,http.StatusInternalServerError, r.Body)
		return
	}
	defer r.Body.Close()

	newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], messsage.BPM)
	if err != nil {
		respondWithJSON(w,r, http.StatusInternalServerError, messsage)
		return
	}
	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
		newBlockChain := append(Blockchain, newBlock)
		replaceChain(newBlockChain)
		spew.Dump(Blockchain)
	}

	respondWithJSON(w,r, http.StatusCreated, newBlock)
}

func respondWithJSON(w http.ResponseWriter, r * http.Request, statusCode int, playload interface{}) {
	response, err := json.MarshalIndent(playload, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	w.WriteHeader(statusCode)
	w.Write(response)
}