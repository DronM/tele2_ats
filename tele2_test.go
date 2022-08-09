package tele2_ats

import(
	"fmt"
	"testing"
	"encoding/json"
	"io/ioutil"
	"bytes"	
	"os"
	"time"
)

const CONF_FILE = "tele2.json"

func FileExists(fileName string) bool {
	if _, err := os.Stat(fileName); err == nil || !os.IsNotExist(err) {
		return true
	}
	return false
}

func PrintStruct(str interface{}) error {
	res, err := json.Marshal(str)
	if err != nil {
		return err
	}
	fmt.Println(string(res))
	return nil
}


func ReadConf(fileName string, tele2 *Tele2Ats) error{
	file, err := ioutil.ReadFile(fileName)
	if err == nil {
		file = bytes.TrimPrefix(file, []byte("\xef\xbb\xbf"))
		err = json.Unmarshal([]byte(file), tele2)		
	}
	if err == nil {
		tele2.AccessTokenDuration = time.Hour * time.Duration(24)
		tele2.RefreshTokenDuration = time.Hour * time.Duration(168)
	}
	return err
}

/*
func TestGetEmployees(t *testing.T) {
	if !FileExists(CONF_FILE) {
		t.Fatalf("Config file %s not found!", CONF_FILE)
	}
	
	tele2 := Tele2Ats{}
	if err := ReadConf(CONF_FILE, &tele2); err != nil {
		t.Fatalf("%v", err)
	}
	
	employees, err := tele2.GetEmployees()
	if err != nil {
		t.Fatalf("%v", err)
	}
	
	if err := PrintStruct(employees); err != nil {
		t.Fatalf("%v", err)
	}
}
*/

/*
func TestActiveCalls(t *testing.T) {
	if !FileExists(CONF_FILE) {
		t.Fatalf("Config file %s not found!", CONF_FILE)
	}
	
	tele2 := Tele2Ats{}
	if err := ReadConf(CONF_FILE, &tele2); err != nil {
		t.Fatalf("%v", err)
	}
	
	calls, err := tele2.GetActiveCalls()
	if err != nil {
		t.Fatalf("%v", err)
	}
	
	if err := PrintStruct(calls); err != nil {
		t.Fatalf("%v", err)
	}
}
*/

func TestWaitForNewCalls(t *testing.T) {
	if !FileExists(CONF_FILE) {
		t.Fatalf("Config file %s not found!", CONF_FILE)
	}
	
	tele2 := Tele2Ats{}
	if err := ReadConf(CONF_FILE, &tele2); err != nil {
		t.Fatalf("%v", err)
	}
	
	for calls := range tele2.WaitForNewCalls(time.Second*2){
		if calls.Error != nil {
			fmt.Printf("Error: %v\n", calls.Error)
		}else{
			fmt.Println("*** Active calls ")
			PrintStruct(calls.Calls)
		}
	}
}

