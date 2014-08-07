// msggw
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nu7hatch/gouuid"
	"os"
	"os/exec"
	"regexp"
	"strconv"
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
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Failed to send:", err)
		}
	}()
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
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Failed to send:", err)
		}
	}()
	sms := getAllSms()
	for key, v := range sms {
		msgId, _ := uuid.NewV4()
		queryDb(ds, `INSERT INTO messages SET ID=?,SENDER=?,SENDER_CODE=?,SENDER_NAME=?,
		RECEIVER=?,RECEIVER_CODE=?,RECEIVER_NAME=?,SUBJECT=?,BODY=?,TIME_CREATED=?,HAS_READ=0,
		PROPERTIES='{}',CORRELATION_ID=''`, msgId.String(), "-1", "syhstem", "系统",
			"1184785174974", "FS0001", "福沙科技", "MSG_UP", v, time.Now())

		command := fmt.Sprint("/usr/bin/gammu deletesms 1 ", key)
		out, err := exec.Command("sh", "-c", command).Output()
		if err != nil {
			fmt.Println("Failed to execute:", err)
		}
		fmt.Printf("%s\n", out)
	}

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

var getAllSms = func() map[int]string {
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
	data := fmt.Sprintf("%s\n", out)
	return splitUpSms(data)
}

var splitUpSms = func(s string) map[int]string {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Failed to send:", err)
		}
	}()
	ret := make(map[int]string)

	regReplace := regexp.MustCompile("(?m)^\\d+ SMS parts in \\d+ SMS sequences$")
	s = regReplace.ReplaceAllString(s, "")

	reg := regexp.MustCompile("(?m)^Location\\s")
	values := reg.Split(s, -1)

	for _, v := range values {
		v = strings.TrimSpace(v)
		if len(v) > 0 {
			key := captureSmsLocation(v)
			reg := regexp.MustCompile("^(?m)\\d+, folder \"Inbox\", SIM memory, Inbox folder$")
			v = reg.ReplaceAllString(v, "")
			v = strings.TrimSpace(v)
			ret[key] = v
		}
	}
	return ret
}

var captureSmsLocation = func(s string) int {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Failed to send:", err)
		}
	}()
	reg := regexp.MustCompile("^\\d+")
	values := reg.FindAllString(s, -1)
	ret, _ := strconv.Atoi(values[0])
	return ret
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
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Failed to send:", err)
		}
	}()
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
		_, err := db.Exec(sqlStatement, sqlParams...)
		if err != nil {
			fmt.Println(err)
		}
	}
	return
}
