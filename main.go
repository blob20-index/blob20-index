package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/glebarez/sqlite"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	text_end_block = 19577001
	api_url        = "https://api.ethscriptions.com/v2"
	eth_rpc        = "https://rpc.ankr.com/eth"
)

type Config struct {
	HttpServer  bool   `json:"http"`
	HttpPort    int    `json:"http_port"`
	HttpsServer bool   `json:"https"`
	HttpsPort   int    `json:"https_port"`
	CertFile    string `json:"cert_file"`
	KeyFile     string `json:"key_file"`
	SqlType     string `json:"sql_type"`
	Database    string `json:"database"`
}

var config Config
var sql_struct = "CREATE TABLE IF NOT EXISTS `blob20_account` (`address` TEXT COLLATE NOCASE NOT NULL,`protocol` TEXT COLLATE NOCASE NOT NULL,`ticker` TEXT COLLATE NOCASE NOT NULL,`balance` REAL NOT NULL DEFAULT 0,PRIMARY KEY (`address`, `protocol`, `ticker`));CREATE TABLE IF NOT EXISTS `blob20_deploy` (`transaction_hash` TEXT COLLATE NOCASE NOT NULL,`block_blockhash` TEXT COLLATE NOCASE NOT NULL,`block_number` INTEGER NOT NULL,`block_timestamp` INTEGER NOT NULL,`deployer` TEXT COLLATE NOCASE NOT NULL,`protocol` TEXT COLLATE NOCASE NOT NULL,`ticker` TEXT COLLATE NOCASE NOT NULL,`max_supply` REAL,`max_limit_per_mint` REAL,`decimals` INTEGER,`mint_amount` REAL,`mint_quantity` INTEGER,`mint_start_block` INTEGER,`mint_end_block` INTEGER,PRIMARY KEY (`protocol`, `ticker`));CREATE TABLE IF NOT EXISTS `blob20_record` (`transaction_hash` TEXT COLLATE NOCASE NOT NULL,`block_blockhash` TEXT COLLATE NOCASE NOT NULL,`block_number` INTEGER NOT NULL,`block_timestamp` INTEGER NOT NULL,`index` INTEGER NOT NULL,`protocol` TEXT COLLATE NOCASE NOT NULL,`ticker` TEXT COLLATE NOCASE NOT NULL,`operation` TEXT COLLATE NOCASE NOT NULL,`from` TEXT COLLATE NOCASE NOT NULL,`to` TEXT COLLATE NOCASE NOT NULL,`amount` REAL NOT NULL,`from_before_amount` REAL NOT NULL,`from_after_amount` REAL NOT NULL,`to_before_amount` REAL NOT NULL,`to_after_amount` REAL NOT NULL,`gas_fee` REAL,`status` TEXT COLLATE NOCASE NOT NULL,`status_msg` TEXT COLLATE NOCASE,`remark` TEXT COLLATE NOCASE,PRIMARY KEY (`transaction_hash`, `from`, `to`, `index`));CREATE TABLE IF NOT EXISTS `blob_transactions` (`transaction_hash` TEXT COLLATE NOCASE NOT NULL,`block_number` INTEGER,`transaction_index` INTEGER,`block_timestamp` TEXT COLLATE NOCASE,`block_blockhash` TEXT COLLATE NOCASE,`event_log_index` INTEGER,`ethscription_number` TEXT COLLATE NOCASE,`creator` TEXT COLLATE NOCASE,`initial_owner` TEXT COLLATE NOCASE,`current_owner` TEXT COLLATE NOCASE,`previous_owner` TEXT COLLATE NOCASE,`content_uri` TEXT COLLATE NOCASE,`content_sha` TEXT COLLATE NOCASE,`esip6` INTEGER,`mimetype` TEXT COLLATE NOCASE,`media_type` TEXT COLLATE NOCASE,`mime_subtype` TEXT COLLATE NOCASE,`gas_price` TEXT COLLATE NOCASE,`gas_used` INTEGER,`transaction_fee` TEXT COLLATE NOCASE,`value` TEXT COLLATE NOCASE,`attachment_sha` TEXT COLLATE NOCASE,`attachment_content_type` TEXT COLLATE NOCASE,`attachment_path` TEXT COLLATE NOCASE,`blob20` TEXT COLLATE NOCASE,`blob_gas_price` TEXT COLLATE NOCASE,`blob_gas_used` INTEGER,`blob_gas_fee` TEXT COLLATE NOCASE,`protocol` TEXT COLLATE NOCASE,`ticker` TEXT COLLATE NOCASE,`operation` TEXT COLLATE NOCASE,`amount` REAL,`is_valid` INTEGER,PRIMARY KEY (`transaction_hash`));"

func init() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %s", err)
	}
	configPath := dir + "/config.json"
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("Error parsing config file: %s", err)
	}

	if config.SqlType == "sqlite" {
		db, err := sqlx.Connect(config.SqlType, config.Database)
		if err != nil {
			log.Fatalf("Error parsing sql file: %s", err)
		}
		defer db.Close()

		_, err = db.Exec(sql_struct)
		if err != nil {
			log.Fatalf("Error exec sql: %s", err)
		}
	}

}

func main() {
	configJson, _ := json.Marshal(config)
	log.Printf("config details: %s \n", configJson)

	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %s", err)
	}

	logPath := dir + "/indexer.log"
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	go blob20Indexer()

	if config.HttpServer || config.HttpsServer {
		http.HandleFunc("/", index)
		http.HandleFunc("/api/getAccounts", getAccounts)
		http.HandleFunc("/api/getTickers", getTickers)
		http.HandleFunc("/api/getRecords", getRecords)
	}

	if config.HttpServer {
		go http.ListenAndServe(":"+strconv.Itoa(config.HttpPort), nil)
	}
	if config.HttpsServer {
		go http.ListenAndServeTLS(":"+strconv.Itoa(config.HttpsPort), config.CertFile, config.KeyFile, nil)
	}
	select {}
}

func index(w http.ResponseWriter, r *http.Request) {
	targetURL := "https://github.com/blob20-index/blob20-index"
	http.Redirect(w, r, targetURL, http.StatusMovedPermanently)
	return
}

func redirectHTTPToHTTPS(w http.ResponseWriter, r *http.Request) {
	if config.HttpsServer && r.TLS == nil && !strings.HasPrefix(r.URL.Scheme, "https") {
		targetURL := "https://" + r.Host + r.URL.RequestURI()
		http.Redirect(w, r, targetURL, http.StatusMovedPermanently)
		return
	}
}

func getTickers(w http.ResponseWriter, r *http.Request) {
	redirectHTTPToHTTPS(w, r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	tx := r.URL.Query().Get("tx")
	protocol := r.URL.Query().Get("protocol")
	ticker := r.URL.Query().Get("ticker")

	db, err := sqlx.Connect(config.SqlType, config.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	query := `SELECT * FROM blob20_deploy WHERE 1 = 1`
	args := make([]interface{}, 0)

	if tx != "" {
		query += " AND transaction_hash = ?"
		args = append(args, tx)
	}
	if protocol != "" {
		query += " AND `protocol` = ?"
		args = append(args, protocol)
	}
	if ticker != "" {
		query += " AND `ticker` = ?"
		args = append(args, ticker)
	}
	query += " ORDER BY block_number ASC "

	var deploys []Blob20Deploy
	if db.Select(&deploys, query, args...) != nil {
		log.Println(err)
		http.Error(w, "Database query error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(deploys)
}

func getRecords(w http.ResponseWriter, r *http.Request) {
	redirectHTTPToHTTPS(w, r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	db, err := sqlx.Connect(config.SqlType, config.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	tx := r.URL.Query().Get("tx")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	protocol := r.URL.Query().Get("protocol")
	ticker := r.URL.Query().Get("ticker")
	operation := r.URL.Query().Get("operation")

	query := `SELECT * FROM blob20_record WHERE 1=1`
	args := make([]interface{}, 0)

	if tx != "" {
		query += " AND transaction_hash = ?"
		args = append(args, tx)
	}
	if from != "" {
		query += " AND `from` = ?"
		args = append(args, from)
	}
	if to != "" {
		query += " AND `to` = ?"
		args = append(args, to)
	}
	if protocol != "" {
		query += " AND `protocol` = ?"
		args = append(args, protocol)
	}
	if ticker != "" {
		query += " AND `ticker` = ?"
		args = append(args, ticker)
	}
	if operation != "" {
		query += " AND `operation` = ?"
		args = append(args, operation)
	}

	query += " ORDER BY block_number DESC,`index` DESC LIMIT ?, ?"
	args = append(args, offset, pageSize)

	var records []Blob20Record
	for db.Select(&records, query, args...) != nil {
		log.Println(err)
		http.Error(w, "Failed to scan records", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(records)
}

func getAccounts(w http.ResponseWriter, r *http.Request) {
	redirectHTTPToHTTPS(w, r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	db, err := sqlx.Connect(config.SqlType, config.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	address := r.URL.Query().Get("address")
	protocol := r.URL.Query().Get("protocol")
	ticker := r.URL.Query().Get("ticker")

	query := `SELECT * FROM blob20_account WHERE 1=1`
	args := make([]interface{}, 0)

	if address != "" {
		query += " AND address = ?"
		args = append(args, address)
	}
	if protocol != "" {
		query += " AND protocol = ?"
		args = append(args, protocol)
	}
	if ticker != "" {
		query += " AND ticker = ?"
		args = append(args, ticker)
	}

	query += " ORDER BY `balance` DESC,`ticker` DESC LIMIT ?, ?"
	args = append(args, offset, pageSize)

	var accounts []Blob20Account
	if db.Select(&accounts, query, args...) != nil {
		log.Println(err)
		http.Error(w, "Failed to scan accounts", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(accounts)
}

func createOrQueryAccount(query string, db *sqlx.DB, tx *sqlx.Tx, address string, protocol string, ticker string) Blob20Account {
	query = `SELECT address, protocol, ticker, balance FROM blob20_account WHERE address = ? AND protocol = ? AND ticker = ?`
	row := db.QueryRow(query, address, protocol, ticker)

	var account Blob20Account
	err := row.Scan(&account.Address, &account.Protocol, &account.Ticker, &account.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			query := `INSERT INTO blob20_account(address, protocol, ticker) VALUES(?, ?, ?)`
			_, err := db.Exec(query, address, protocol, ticker)
			if err != nil {
				log.Fatal(err)
				tx.Rollback()
			}
			account.Address = address
			account.Protocol = protocol
			account.Ticker = ticker
		} else {
			log.Fatal(err)
			tx.Rollback()
		}
	}
	return account
}

func IsValidEthereumAddress(address string) bool {
	re := regexp.MustCompile("^0x[a-fA-F0-9]{40}$")
	return re.MatchString(address)
}

func blob20Indexer() {
	db, err := sqlx.Connect(config.SqlType, config.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	client, err := ethclient.Dial(eth_rpc)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	var lastBlock int
	query := "SELECT block_number FROM blob_transactions ORDER BY block_number DESC LIMIT 1"
	err = db.QueryRow(query).Scan(&lastBlock)
	if err != nil {
		log.Printf("init:: get lastest block number error : %v, init start block: 1 \n", err)
		lastBlock = 1
	} else {
		lastBlock = lastBlock - 1
	}
	log.Printf("init:: start block: %d \n", lastBlock)

	pageKey := ""
	latestKey := ""
	for i := 1; true; i++ {
		log.Println("page : " + strconv.Itoa(i) + ", " + pageKey)

		url := api_url + "/ethscriptions?" +
			"&attachment_content_type[]=application/json" +
			"&attachment_content_type[]=text/plain" +
			"&mimetype=text/plain" +
			"&after_block=" + strconv.Itoa(lastBlock) +
			"&sort_by=block_number" +
			"&reverse=true" +
			"&max_results=100" +
			pageKey

		req, _ := http.NewRequest("GET", url, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println("getdata::err 1: " + err.Error())
			log.Fatal(err)
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			log.Println("getdata::err 2: " + err.Error())
			log.Fatal(err)
		}
		resp := new(Resp)
		json.Unmarshal(body, resp)
		log.Println("getdata::pagesize: " + strconv.Itoa(len(resp.Result)))
		if len(resp.Result) == 0 {
			i--
		}
		for _, blob := range resp.Result {
			time.Sleep(10 * time.Millisecond)
			var exists bool
			query := "SELECT EXISTS(SELECT 1 FROM blob_transactions WHERE transaction_hash = ?)"
			err = db.QueryRow(query, blob.TransactionHash).Scan(&exists)
			if err != nil {
				log.Printf("getdata::Error get transaction does error. hash: %v \n", err)
				continue
			}

			if exists {
				log.Println("getdata::exists, continue..：" + blob.TransactionHash)
				continue
			}

			blockNumber, _ := strconv.Atoi(blob.BlockNumber)
			if blob.AttachmentContentType == "application/json" || (blob.AttachmentContentType == "text/plain" && blockNumber <= text_end_block) {

				var attachmentSha, blob20 string
				query := `SELECT attachment_sha, blob20 FROM blob_transactions WHERE attachment_sha = ? LIMIT 1`
				err = db.QueryRow(query, blob.AttachmentSha).Scan(&attachmentSha, &blob20)
				if err != nil || blob20 == "" {
					attUrl := api_url + blob.AttachmentPath
					attrReq, _ := http.NewRequest("GET", attUrl, nil)
					attrRes, err := http.DefaultClient.Do(attrReq)
					if err != nil {
						log.Println("getdata::err 3: " + err.Error())
						log.Fatal(err)
					}
					attrBody, err := io.ReadAll(attrRes.Body)
					attrRes.Body.Close()
					if err != nil {
						log.Println("getdata::err 4: " + err.Error())
						log.Fatal(err)
					}
					blob.Blob20 = string(bytes.TrimPrefix(attrBody, []byte("\xef\xbb\xbf")))
				} else {
					blob.Blob20 = blob20
				}
				blob.Blob20 = strings.ToLower(blob.Blob20)

				protocol := new(Protocol)
				err = json.Unmarshal([]byte(blob.Blob20), &protocol)

				blob.Blob20 = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(blob.Blob20, " ", ""), "\r", ""), "\n", ""), "\t", "")

				if err != nil {
					log.Printf("getdata::parsing json : %s \n", blob.Blob20)
					log.Println(err.Error())
					log.Printf("getdata::parsing failed. tx：%s \n", blob.TransactionHash)
				} else {
					receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash(blob.TransactionHash))
					if err != nil {
						log.Println("getdata::err 5: " + err.Error())
						log.Fatal(err)
					}
					if receipt.Type != 3 {
						log.Println(receipt.Type)
						log.Println("getdata:: type is not eip 4844 tx: " + blob.TransactionHash)
						continue
					}

					blobGasFee := decimal.NewFromUint64(receipt.BlobGasPrice.Uint64() * receipt.BlobGasUsed)

					if protocol.Protocol == "blob20" {
						protocol.Token.Ticker = strings.ToUpper(protocol.Token.Ticker)
						switch operation := protocol.Token.Operation; strings.ToLower(operation) {
						case "deploy":
							var exists bool
							query := "SELECT EXISTS(SELECT 1 FROM blob20_deploy WHERE protocol = ? AND ticker = ?)"
							err = db.QueryRow(query, protocol.Protocol, protocol.Token.Ticker).Scan(&exists)
							if err != nil {
								log.Printf("%v", err)
								break
							}

							if exists {
								log.Printf("deploy:: ticker exists, hash: %s, protocol: %s, ticker: %s \n", blob.TransactionHash, protocol.Protocol, protocol.Token.Ticker)
								break
							} else {
								tx, err := db.Beginx()
								if err != nil {
									log.Fatalln(err)
								}

								_, err = db.Exec("INSERT INTO blob_transactions (transaction_hash, block_number, transaction_index, block_timestamp, block_blockhash, event_log_index, ethscription_number, creator, initial_owner, current_owner, previous_owner, content_uri, content_sha, esip6, mimetype, media_type, mime_subtype, gas_price, gas_used, transaction_fee, value, attachment_sha, attachment_content_type, attachment_path, is_valid, blob20, blob_gas_price, blob_gas_used, blob_gas_fee) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
									blob.TransactionHash, blob.BlockNumber, blob.TransactionIndex, blob.BlockTimestamp, blob.BlockBlockhash, blob.EventLogIndex, blob.EthscriptionNumber, blob.Creator, blob.InitialOwner, blob.CurrentOwner, blob.PreviousOwner, blob.ContentURI, blob.ContentSha, blob.Esip6, blob.Mimetype, blob.MediaType, blob.MimeSubtype, blob.GasPrice, blob.GasUsed, blob.TransactionFee, blob.Value, blob.AttachmentSha, blob.AttachmentContentType, blob.AttachmentPath, false, blob.Blob20, receipt.BlobGasPrice.String(), decimal.NewFromUint64(receipt.BlobGasUsed).String(), blobGasFee)
								if err != nil {
									log.Printf("deploy:: Error inserting transactions data: %v \n", err)
									tx.Rollback()
									break
								}

								if protocol.Token.MaxSupply.LessThanOrEqual(decimal.NewFromInt(0)) &&
									protocol.Token.Supply.GreaterThan(decimal.NewFromInt(0)) {
									protocol.Token.MaxSupply = protocol.Token.Supply
								}
								if protocol.Token.MaxLimitPerMint.LessThanOrEqual(decimal.NewFromInt(0)) &&
									protocol.Token.Limit.GreaterThan(decimal.NewFromInt(0)) {
									protocol.Token.MaxLimitPerMint = protocol.Token.Limit
								}

								if protocol.Token.Ticker != "" && protocol.Token.MaxSupply.GreaterThan(decimal.NewFromInt(0)) && protocol.Token.MaxLimitPerMint.GreaterThan(decimal.NewFromInt(0)) && protocol.Token.Decimals >= 0 {
									updateSql := "UPDATE blob_transactions SET protocol = ?, ticker = ?, operation = ?, is_valid = ? WHERE transaction_hash = ?"
									_, err := db.Exec(updateSql, protocol.Protocol, protocol.Token.Ticker, protocol.Token.Operation, true, blob.TransactionHash)
									if err != nil {
										log.Printf("deploy:: update blob transactions error: %v", err)
										tx.Rollback()
										break
									} else {
										insertDeploy := `INSERT INTO blob20_deploy(transaction_hash, block_blockhash, block_number, block_timestamp, deployer, protocol, ticker, max_supply, max_limit_per_mint, decimals, mint_amount, mint_quantity, mint_start_block, mint_end_block) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`
										_, err := db.Exec(insertDeploy, blob.TransactionHash, blob.BlockBlockhash, blob.BlockNumber, blob.BlockTimestamp, blob.Creator, protocol.Protocol, protocol.Token.Ticker, protocol.Token.MaxSupply, protocol.Token.MaxLimitPerMint, protocol.Token.Decimals, 0, 0, blob.BlockNumber, 0)
										if err != nil {
											log.Printf("deploy::Error inserting new deploy record: %v \n hash: %s \n", err, blob.TransactionHash)
											tx.Rollback()
										} else {
											log.Printf("deploy::deploy inserted successfully! ticker: %s, hash：%s \n", protocol.Token.Ticker, blob.TransactionHash)
											tx.Commit()
										}
									}
								}
							}
						case "mint":
							var exists bool
							query := "SELECT EXISTS(SELECT 1 FROM blob20_deploy WHERE protocol = ? AND ticker = ?)"
							err := db.QueryRow(query, protocol.Protocol, protocol.Token.Ticker).Scan(&exists)
							if err != nil {
								log.Printf("mint:: 1 error: %v \n", err)
							}

							if !exists {
								log.Printf("mint:: ticker does not exist, protocol: %s, ticker: %s, hash: %s \n", protocol.Protocol, protocol.Token.Ticker, blob.TransactionHash)
								break
							} else {
								query := `SELECT * FROM blob20_deploy WHERE protocol = ? AND ticker = ? LIMIT 1;`
								var deploy Blob20Deploy
								err = db.Get(&deploy, query, protocol.Protocol, protocol.Token.Ticker)

								if err != nil {
									if err == sql.ErrNoRows {
										log.Printf("mint::no found record, protocol: %s , token: %s .\n", protocol.Protocol, protocol.Token.Ticker)
									} else {
										log.Printf("mint::no found record, error: %v \n", err)
									}
									break
								}

								if deploy.MintAmount.GreaterThanOrEqual(deploy.MaxSupply) {
									log.Printf("mint:: ticker is minted. ticker: %s, amount: %s, supply: %s, mint is completed.. \n", deploy.Ticker, deploy.MintAmount, deploy.MaxSupply)
									break
								}

								if protocol.Token.Amount.GreaterThan(deploy.MaxLimitPerMint) {
									log.Printf("mint:: ticker mint exceeding limit, mint ticker: %s, amount: %s, limit: %s, hash: %s \n", protocol.Token.Ticker, protocol.Token.Amount, deploy.MaxLimitPerMint, blob.TransactionHash)
									break
								}

								if protocol.Token.Amount.LessThanOrEqual(deploy.MaxLimitPerMint) && blockNumber > deploy.MintStartBlock && deploy.MintAmount.LessThan(deploy.MaxSupply) {
									tx, err := db.Beginx()
									if err != nil {
										log.Fatalln(err)
									}
									_, err = db.Exec("INSERT INTO blob_transactions (transaction_hash, block_number, transaction_index, block_timestamp, block_blockhash, event_log_index, ethscription_number, creator, initial_owner, current_owner, previous_owner, content_uri, content_sha, esip6, mimetype, media_type, mime_subtype, gas_price, gas_used, transaction_fee, value, attachment_sha, attachment_content_type, attachment_path, is_valid, blob20, blob_gas_price, blob_gas_used, blob_gas_fee) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
										blob.TransactionHash, blob.BlockNumber, blob.TransactionIndex, blob.BlockTimestamp, blob.BlockBlockhash, blob.EventLogIndex, blob.EthscriptionNumber, blob.Creator, blob.InitialOwner, blob.CurrentOwner, blob.PreviousOwner, blob.ContentURI, blob.ContentSha, blob.Esip6, blob.Mimetype, blob.MediaType, blob.MimeSubtype, blob.GasPrice, blob.GasUsed, blob.TransactionFee, blob.Value, blob.AttachmentSha, blob.AttachmentContentType, blob.AttachmentPath, false, blob.Blob20, receipt.BlobGasPrice.String(), decimal.NewFromUint64(receipt.BlobGasUsed).String(), blobGasFee)
									if err != nil {
										log.Printf("mint:: Error inserting transactions data: %v \n", err)
										tx.Rollback()
										break
									}

									var exists bool
									query = "SELECT EXISTS(SELECT 1 FROM blob20_record WHERE transaction_hash = ?)"

									err = db.QueryRow(query, blob.TransactionHash).Scan(&exists)
									if err != nil {
										log.Printf("mint:: 2 error: %v \n", err)
										break
									}

									if exists {
										log.Println("mint:: record exists, continue..：" + blob.TransactionHash)
										break
									}

									account := createOrQueryAccount(query, db, tx, blob.InitialOwner, deploy.Protocol, deploy.Ticker)

									mintEndBlock := 0
									remainingAmount := deploy.MaxSupply.Sub(deploy.MintAmount)
									if remainingAmount.LessThanOrEqual(protocol.Token.Amount) {
										protocol.Token.Amount = remainingAmount
										mintEndBlock = blockNumber
									}

									//insert mint record
									query = "INSERT INTO blob20_record(transaction_hash, block_blockhash, block_number, block_timestamp, `index`, protocol, ticker, operation, `from`, `to`, amount, from_before_amount, from_after_amount, to_before_amount, to_after_amount, gas_fee, `status`, status_msg, remark) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
									_, err = db.Exec(query, blob.TransactionHash, blob.BlockBlockhash, blob.BlockNumber, blob.BlockTimestamp, blob.TransactionIndex, deploy.Protocol, deploy.Ticker, protocol.Token.Operation, blob.Creator, blob.InitialOwner, protocol.Token.Amount, 0.00, 0.00, account.Balance, protocol.Token.Amount.Add(account.Balance), blob.TransactionFee.Add(blobGasFee).Div(decimal.NewFromInt(1e18)), "success", nil, nil)

									if err != nil {
										log.Printf("mint:: insert mint record fail, hash: %s, error: %v \n.", blob.TransactionHash, err)
										tx.Rollback()
										break
									} else {
										//update amount and quantity
										query := `UPDATE blob20_deploy SET mint_amount = ?, mint_quantity = ?, mint_end_block = ? WHERE protocol = ? AND ticker = ?`
										_, err := db.Exec(query, deploy.MintAmount.Add(protocol.Token.Amount), deploy.MintQuantity+1, mintEndBlock, deploy.Protocol, deploy.Ticker)
										if err != nil {
											log.Printf("mint:: 4 error: %v \n", err)
											tx.Rollback()
											break
										}

										//update balance
										query = "UPDATE blob20_account SET balance = ? WHERE `address` = ? AND protocol = ? AND ticker = ?"
										_, err = db.Exec(query, protocol.Token.Amount.Add(account.Balance), blob.InitialOwner, account.Protocol, account.Ticker)
										if err != nil {
											log.Printf("mint:: 6 error: %v \n", err)
											tx.Rollback()
											break
										}

										updateSql := "UPDATE blob_transactions SET protocol = ?, ticker = ?, operation = ?, amount = ?, is_valid = ? WHERE transaction_hash = ?"
										_, err = db.Exec(updateSql, deploy.Protocol, deploy.Ticker, protocol.Token.Operation, protocol.Token.Amount, true, blob.TransactionHash)
										if err != nil {
											log.Printf("mint:: update blob transactions error: %v \n", err)
											tx.Rollback()
											break
										}
										tx.Commit()
										log.Printf("mint::mint inserted successfully! ticker: %s, amount: %s, hash：%s \n", protocol.Token.Ticker, protocol.Token.Amount, blob.TransactionHash)
									}
									tx.Commit()
								} else {
									log.Printf("mint:: verification failed, hash: %s \n", blob.TransactionHash)
									continue
								}
							}
						case "transfer":
							var exists bool
							query := "SELECT EXISTS(SELECT 1 FROM blob20_deploy WHERE protocol = ? AND ticker = ?)"
							err := db.QueryRow(query, protocol.Protocol, protocol.Token.Ticker).Scan(&exists)
							if err != nil {
								log.Printf("transfer:: 1 error: %v \n", err)
							}

							if !exists {
								log.Printf("transfer:: ticker does not exist, hash: %s, protocol: %s, ticker: %s \n", blob.TransactionHash, protocol.Protocol, protocol.Token.Ticker)
								break
							} else {

								query := `SELECT * FROM blob20_deploy WHERE protocol = ? AND ticker = ? LIMIT 1;`
								var deploy Blob20Deploy
								err = db.Get(&deploy, query, protocol.Protocol, protocol.Token.Ticker)

								if err != nil {
									if err == sql.ErrNoRows {
										log.Printf("transfer::no found record, protocol: %s , token: %s .\n", protocol.Protocol, protocol.Token.Ticker)
									} else {
										log.Printf("transfer::no found record, error: %v \n", err)
									}
									break
								} else {
									if blob.Creator == blob.InitialOwner {
										amount := decimal.NewFromFloat(0.00)
										flag := false
										for _, transfer := range protocol.Token.Transfers {
											if transfer.To == "" && IsValidEthereumAddress(transfer.ToAddress) {
												transfer.To = transfer.ToAddress
											}
											if !IsValidEthereumAddress(transfer.To) {
												log.Printf("trasfer::address verification failed：to: %s, hash: %s \n", transfer.To, blob.TransactionHash)
												flag = true
												break
											}
											if transfer.Amount.LessThanOrEqual(decimal.NewFromInt(0)) {
												log.Printf("trasfer::amount verification failed：amount: %s, hash: %s \n", transfer.Amount, blob.TransactionHash)
												flag = true
												break
											}
											amount = amount.Add(transfer.Amount)
										}
										if flag {
											break
										}

										var exists bool
										query = "SELECT EXISTS(SELECT 1 FROM blob20_record WHERE transaction_hash = ?)"

										err = db.QueryRow(query, blob.TransactionHash).Scan(&exists)
										if err != nil {
											log.Printf("transfer:: 2 error: %v \n", err)
										}

										if exists {
											log.Println("trasfer:: record exists, continue..：" + blob.TransactionHash)
											continue
										}

										tx, err := db.Beginx()
										if err != nil {
											log.Fatalln(err)
										}
										_, err = db.Exec("INSERT INTO blob_transactions (transaction_hash, block_number, transaction_index, block_timestamp, block_blockhash, event_log_index, ethscription_number, creator, initial_owner, current_owner, previous_owner, content_uri, content_sha, esip6, mimetype, media_type, mime_subtype, gas_price, gas_used, transaction_fee, value, attachment_sha, attachment_content_type, attachment_path, is_valid, blob20, blob_gas_price, blob_gas_used, blob_gas_fee) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
											blob.TransactionHash, blob.BlockNumber, blob.TransactionIndex, blob.BlockTimestamp, blob.BlockBlockhash, blob.EventLogIndex, blob.EthscriptionNumber, blob.Creator, blob.InitialOwner, blob.CurrentOwner, blob.PreviousOwner, blob.ContentURI, blob.ContentSha, blob.Esip6, blob.Mimetype, blob.MediaType, blob.MimeSubtype, blob.GasPrice, blob.GasUsed, blob.TransactionFee, blob.Value, blob.AttachmentSha, blob.AttachmentContentType, blob.AttachmentPath, false, blob.Blob20, receipt.BlobGasPrice.String(), decimal.NewFromUint64(receipt.BlobGasUsed).String(), blobGasFee)
										if err != nil {
											log.Printf("transfer:: Error inserting transactions data: %v \n", err)
											tx.Rollback()
											break
										}

										account := createOrQueryAccount(query, db, tx, blob.InitialOwner, deploy.Protocol, deploy.Ticker)
										if account.Balance.GreaterThanOrEqual(amount) {

											for i, transfer := range protocol.Token.Transfers {
												if transfer.To == "" && IsValidEthereumAddress(transfer.ToAddress) {
													transfer.To = transfer.ToAddress
												}

												toAccount := createOrQueryAccount(query, db, tx, transfer.To, deploy.Protocol, deploy.Ticker)
												if account.Address == toAccount.Address {
													toAccount.Balance = toAccount.Balance.Sub(transfer.Amount)
												}

												query = "INSERT INTO blob20_record(transaction_hash, block_blockhash, block_number, block_timestamp, `index`, protocol, ticker, operation, `from`, `to`, amount, from_before_amount, from_after_amount, to_before_amount, to_after_amount, gas_fee, `status`, status_msg, remark) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
												_, err = db.Exec(query, blob.TransactionHash, blob.BlockBlockhash, blob.BlockNumber, blob.BlockTimestamp, i, deploy.Protocol, deploy.Ticker, protocol.Token.Operation, blob.Creator, transfer.To, transfer.Amount, account.Balance, account.Balance.Sub(transfer.Amount), toAccount.Balance, toAccount.Balance.Add(transfer.Amount), blob.TransactionFee.Add(blob.BlobGasFee).Div(decimal.NewFromInt(1e18)), "success", nil, nil)
												account.Balance = account.Balance.Sub(transfer.Amount)
												toAccount.Balance = toAccount.Balance.Add(transfer.Amount)
												if err != nil {
													log.Printf("transfer:: 4 error: %v \n", err)
													tx.Rollback()
													break
												} else {
													//update balance
													query = "UPDATE blob20_account SET balance = ? WHERE `address` = ? AND protocol = ? AND ticker = ?"
													_, err = db.Exec(query, account.Balance, account.Address, account.Protocol, account.Ticker)
													if err != nil {
														log.Printf("transfer:: 5 error: %v \n", err)
														tx.Rollback()
														break
													} else {
														query = "UPDATE blob20_account SET balance = ? WHERE `address` = ? AND protocol = ? AND ticker = ?"
														_, err = db.Exec(query, toAccount.Balance, toAccount.Address, toAccount.Protocol, toAccount.Ticker)
														if err != nil {
															log.Printf("transfer:: 6 error: %v \n", err)
															tx.Rollback()
															break
														} else {
															updateSql := "UPDATE blob_transactions SET protocol = ?, ticker = ?, operation = ?, amount = ?, is_valid = ? WHERE transaction_hash = ?"
															_, err = db.Exec(updateSql, deploy.Protocol, deploy.Ticker, protocol.Token.Operation, amount, true, blob.TransactionHash)
															if err != nil {
																log.Printf("transfer:: update blob transactions error: %v \n", err)
																tx.Rollback()
																break
															} else {
																log.Printf("trasfer:: ticker: %s, from: %s, to: %s, amount: %s :: success", deploy.Ticker, account.Address, toAccount.Address, transfer.Amount)
																tx.Commit()
															}
														}
													}
												}
											}

										} else {
											for i, transfer := range protocol.Token.Transfers {
												toAccount := createOrQueryAccount(query, db, nil, transfer.To, protocol.Protocol, protocol.Token.Ticker)
												query = "INSERT INTO blob20_record(transaction_hash, block_blockhash, block_number, block_timestamp, `index`, protocol, ticker, operation, `from`, `to`, amount, from_before_amount, from_after_amount, to_before_amount, to_after_amount, gas_fee, `status`, status_msg, remark) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
												_, err = db.Exec(query, blob.TransactionHash, blob.BlockBlockhash, blob.BlockNumber, blob.BlockTimestamp, i, protocol.Protocol, protocol.Token.Ticker, protocol.Token.Operation, blob.Creator, transfer.To, transfer.Amount, account.Balance, account.Balance, toAccount.Balance, toAccount.Balance, blob.TransactionFee.Add(blob.BlobGasFee).Div(decimal.NewFromInt(1e18)), "fail", "Insufficient balance", nil)
												if err != nil {
													log.Printf("transfer:: 7 error: %v \n", err)
													tx.Rollback()
													break
												}
											}
										}
										tx.Commit()
									} else {
										log.Println("transfer::not transfer BLOB ：" + blob.TransactionHash)
									}
								}
							}
						case "premine":
						default:
							log.Printf("operation:: Unrecognized operation type: %s, hash: %s \n", operation, blob.TransactionHash)
						}
					} else {
						log.Println("protocol:: is not blob20 ：" + blob.TransactionHash)
					}
				}
			}
			latestKey = blob.TransactionHash
		}
		pageKey = "&page_key=" + resp.Pagination.PageKey
		if !resp.Pagination.HasMore {
			pageKey = "&page_key=" + latestKey
			time.Sleep(10 * time.Second)
		}
	}
}

type Resp struct {
	Result     []Blobs    `json:"result"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	PageKey string `json:"page_key"`
	HasMore bool   `json:"has_more"`
}

type Blobs struct {
	TransactionHash       string          `db:"transaction_hash" json:"transaction_hash"`
	BlockNumber           string          `db:"block_number" json:"block_number"`
	TransactionIndex      string          `db:"transaction_index" json:"transaction_index"`
	BlockTimestamp        string          `db:"block_timestamp" json:"block_timestamp"`
	BlockBlockhash        string          `db:"block_blockhash" json:"block_blockhash"`
	EventLogIndex         *int            `db:"event_log_index" json:"event_log_index"`
	EthscriptionNumber    string          `db:"ethscription_number" json:"ethscription_number"`
	Creator               string          `db:"creator" json:"creator"`
	InitialOwner          string          `db:"initial_owner" json:"initial_owner"`
	CurrentOwner          string          `db:"current_owner" json:"current_owner"`
	PreviousOwner         string          `db:"previous_owner" json:"previous_owner"`
	ContentURI            string          `db:"content_uri" json:"content_uri"`
	ContentSha            string          `db:"content_sha" json:"content_sha"`
	Esip6                 bool            `db:"esip6" json:"esip6"`
	Mimetype              string          `db:"mimetype" json:"mimetype"`
	MediaType             string          `db:"media_type" json:"media_type"`
	MimeSubtype           string          `db:"mime_subtype" json:"mime_subtype"`
	GasPrice              decimal.Decimal `db:"gas_price" json:"gas_price"`
	GasUsed               string          `db:"gas_used" json:"gas_used"`
	TransactionFee        decimal.Decimal `db:"transaction_fee" json:"transaction_fee"`
	Value                 string          `db:"value" json:"value"`
	AttachmentSha         string          `db:"attachment_sha" json:"attachment_sha"`
	AttachmentContentType string          `db:"attachment_content_type" json:"attachment_content_type"`
	AttachmentPath        string          `db:"attachment_path" json:"attachment_path"`
	Blob20                string          `db:"blob20" json:"blob20"`
	BlobGasPrice          decimal.Decimal `db:"blob_gas_price" json:"blob_gas_price,omitempty"`
	BlobGasUsed           *int            `db:"blob_gas_used" json:"blob_gas_used,omitempty"`
	BlobGasFee            decimal.Decimal `db:"blob_gas_fee" json:"blob_gas_fee,omitempty"`
	Protocol              string          `db:"protocol" json:"protocol,omitempty"`
	Ticker                string          `db:"ticker" json:"ticker,omitempty"`
	Operation             string          `db:"operation" json:"operation,omitempty"`
	Amount                decimal.Decimal `db:"amount" json:"amount,omitempty"`
	IsValid               bool            `db:"is_valid" json:"is_valid,omitempty"`
}

type Protocol struct {
	Protocol string `json:"protocol"`
	Token    Token  `json:"token"`
}

type Transfer struct {
	To        string          `json:"to"`
	ToAddress string          `json:"to_address"`
	Amount    decimal.Decimal `json:"amount"`
}

type Token struct {
	Operation       string          `json:"operation"`
	Ticker          string          `json:"ticker"`
	MaxSupply       decimal.Decimal `json:"max_supply"`
	Supply          decimal.Decimal `json:"supply"`
	MaxLimitPerMint decimal.Decimal `json:"max_limit_per_mint"`
	Limit           decimal.Decimal `json:"limit"`
	Decimals        int             `json:"decimals"`
	Amount          decimal.Decimal `json:"amount"`
	Transfers       []Transfer      `json:"transfers"`
}

type Blob20Account struct {
	Address  string          `json:"address" db:"address"`
	Protocol string          `json:"protocol" db:"protocol"`
	Ticker   string          `json:"ticker" db:"ticker"`
	Balance  decimal.Decimal `json:"balance" db:"balance"`
}

type Blob20Record struct {
	TransactionHash  string          `json:"transaction_hash" db:"transaction_hash"`
	BlockBlockhash   string          `json:"block_blockhash" db:"block_blockhash"`
	BlockNumber      int             `json:"block_number" db:"block_number"`
	BlockTimestamp   int             `json:"block_timestamp" db:"block_timestamp"`
	Index            int             `json:"index" db:"index"`
	Protocol         string          `json:"protocol" db:"protocol"`
	Ticker           string          `json:"ticker" db:"ticker"`
	Operation        string          `json:"operation" db:"operation"`
	From             string          `json:"from" db:"from"`
	To               string          `json:"to" db:"to"`
	Amount           decimal.Decimal `json:"amount" db:"amount"`
	FromBeforeAmount decimal.Decimal `json:"from_before_amount" db:"from_before_amount"`
	FromAfterAmount  decimal.Decimal `json:"from_after_amount" db:"from_after_amount"`
	ToBeforeAmount   decimal.Decimal `json:"to_before_amount" db:"to_before_amount"`
	ToAfterAmount    decimal.Decimal `json:"to_after_amount" db:"to_after_amount"`
	GasFee           decimal.Decimal `json:"gas_fee,omitempty" db:"gas_fee"`
	Status           string          `json:"status" db:"status"`
	StatusMsg        *string         `json:"status_msg,omitempty" db:"status_msg"`
	Remark           *string         `json:"remark,omitempty" db:"remark"`
}

type Blob20Deploy struct {
	TransactionHash string          `db:"transaction_hash"`
	BlockBlockhash  string          `db:"block_blockhash"`
	BlockNumber     int             `db:"block_number"`
	BlockTimestamp  int             `db:"block_timestamp"`
	Deployer        string          `db:"deployer"`
	Protocol        string          `db:"protocol"`
	Ticker          string          `db:"ticker"`
	MaxSupply       decimal.Decimal `db:"max_supply"`
	MaxLimitPerMint decimal.Decimal `db:"max_limit_per_mint"`
	Decimals        int             `db:"decimals"`
	MintAmount      decimal.Decimal `db:"mint_amount"`
	MintQuantity    int             `db:"mint_quantity"`
	MintStartBlock  int             `db:"mint_start_block"`
	MintEndBlock    int             `db:"mint_end_block"`
}
