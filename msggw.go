// msggw
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

func main() {
	args := args()
	ds := args[0]
	for {
		workDown(ds)
		workUp(ds)
		time.Sleep(time.Second)
	}

}

var workDown = func(ds string) {
	sqlSelect := `SELECT ID,BODY,PROPERTIES FROM messages 
	WHERE SENDER=? AND RECEIVER=? AND SUBJECT=? AND HAS_READ=? LIMIT 1`
	msg := queryDb(ds, sqlSelect, "-1", "-1", "MSG_DOWN", 0)
	var properties map[string]interface{}
	msgId := msg[0]
	if len(msgId) == 0 {
		return
	}
	msgBody := msg[1]
	msgProperties := msg[2]
	json.Unmarshal([]byte(msgProperties), &properties)
	receivers := properties["receivers"]
	receiverArray, _ := receivers.([]interface{})
	for _, phoneNumber := range receiverArray {
		sendSms(phoneNumber.(string), msgBody)
	}
	queryDb(ds, `UPDATE messages SET HAS_READ=HAS_READ+1 WHERE HAS_READ=0 AND ID=?`, msgId)
}

var workUp = func(ds string) {

}

var sendSms = func(phoneNumber string, message string) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Failed to send:", err)
		}
	}()
	if utf8.RuneCountInString(phoneNumber) != 11 {
		fmt.Println("Phone number with invalid length.")
		return
	}
	if !strings.HasPrefix(phoneNumber, "+") {
		phoneNumber = fmt.Sprint("+86", phoneNumber)
	}
	message = strings.Replace(message, "\"", "\\\"", -1)
	fmt.Println(time.Now(), phoneNumber, message)
	command := fmt.Sprint("/usr/bin/gammu sendsms TEXT ", phoneNumber, " -unicode -text \"", message, "\"")
	out, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		fmt.Println("Failed to execute:", err)
	}
	fmt.Printf("%s\n", out)
}

var getAllSms = func() []string {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Failed to send:", err)
		}
	}()

	command := fmt.Sprint("/usr/bin/gammu getallsms")
	out, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		fmt.Println("Failed to execute:", err)
	}
	data := fmt.SPrintf("%s\n", out)
	fmt.Println(data)
}

var getConn = func(ds string) (*sql.DB, error) {
	return sql.Open("mysql", ds)
}

func args() []string {
	ret := []string{}
	if len(os.Args) != 2 {
		fmt.Println("Usage: msggw ds")
		os.Exit(1)
	} else {
		for i := 1; i < len(os.Args); i++ {
			ret = append(ret, os.Args[i])
		}
	}
	return ret
}

var queryDb = func(ds string, sqlStatement string, sqlParams ...interface{}) (result []string) {
	db, _ := getConn(ds)
	defer db.Close()

	if strings.HasPrefix(strings.ToUpper(sqlStatement), "SELECT") {
		rows, _ := db.Query(sqlStatement, sqlParams...)
		cols, _ := rows.Columns()
		rawResult := make([][]byte, len(cols))
		result = make([]string, len(cols))
		dest := make([]interface{}, len(cols)) // A temporary interface{} slice
		for i, _ := range rawResult {
			dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
		}

		if rows.Next() {
			rows.Scan(dest...)
			for i, raw := range rawResult {
				if raw == nil {
					result[i] = "\\N"
				} else {
					result[i] = string(raw)
				}
			}
		}
	} else {
		db.Exec(sqlStatement, sqlParams...)
	}
	return
}
