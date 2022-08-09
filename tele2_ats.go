package tele2_ats

import(
	"net/http"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"time"
	"errors"
)

const (
	API_URL = "https://ats2.tele2.ru/crm/openapi"
	
	CALL_TYPE_SINGLE_CHANNEL = "SINGLE_CHANNEL"
	CALL_TYPE_OUTGOING = "OUTGOING"
	CALL_TYPE_MULTI_CHANNEL = "MULTI_CHANNEL"
	CALL_TYPE_INTERNAL = "INTERNAL"	
)

const (
	CALL_ATS_START = iota
	CALL_ATS_END
)

const ERR_REFRESH_TOKEN_EXPIRED = "refresh token has expired"

type Tele2Ats struct {
	Login string `json:"login"`
	RegionID int `json:"regionID"`
	CompanyID int `json:"companyID"`
	AccessToken string `json:"accessToken"`	
	AccessTokenDuration time.Duration
	RefreshToken string `json:"refreshToken"`
	RefreshTokenDuration time.Duration
	TokenDate time.Time `json:"tokenDate"`
}

//Структура сотрудников
type Employee struct {
	EmployeeId int `json:"employeeId"`
	Name string `json:"name"`
	Surname string `json:"surname"`
	FullNumber string `json:"fullNumber"`
	GroupName string `json:"groupName"`
	ShortNumber string `json:"shortNumber"`
	Email string `json:"email"`
}

//
type CallInf struct {
	CallType string `json:"callType"`
	CallerNumberFull string `json:"callerNumberFull"`
	CallerNumberShort string `json:"callerNumberShort"`
	CalledNumberFull string `json:"calledNumberFull"`
	CalledNumberShort string `json:"calledNumberShort"`
}

type ActiveCallInf struct {
	Crc uint32
	CallInf	
}

type ActiveCall struct {	
	CallAction int `json:"callAction"`
	CallInf
}

type ActiveCallList map[uint32]ActiveCall

type ActiveCalls struct {
	Calls []ActiveCall//ActiveCallList
	Error error
	AccessToken string
	RefreshToken string
	TokenDate time.Time
}

type CallRecordCalleePart struct {
	BreakTimer int `json:"breakTimer"`
	CallForwardType string `json:"callForwardType"`
	ClientType string `json:"clientType"`
	ClirType string `json:"clirType"`
	ClirTypePbxNumber string `json:"clirTypePbxNumber"` //struct
	Client string `json:"client"` //struct
	ClientForward string `json:"clientForward"` //struct
	ClientForwardOffTime string `json:"clientForwardOffTime"` //struct
	CompanyId int `json:"companyId"`
	Id int `json:"id"`
	IsActive bool `json:"isActive"`
	IvrMenuForward string `json:"ivrMenuForward"` //struct
	IvrMenuForwardOffTime string `json:"ivrMenuForwardOffTime"` //struct
	Number string `json:"number"`
	NumberType string `json:"numberType"`
	QueueForward string `json:"queueForward"` //struct
	QueueForwardOffTime string `json:"queueForwardOffTime"` //struct
}

type CallRecord struct {
	CallDate time.Time `json:"callDate"`
	CallDuration int `json:"callDuration"`
	RecordName string `json:"recordName"`
	CalleePart CallRecordCalleePart `json:"calleePart"` 
	CallerPart CallRecordCalleePart `json:"callerPart"` 
	Id int `json:"id"`
	//
	CallStatus string `json:"callStatus"`
	CallTimestamp int `json:"callTimestamp"`
	CallType string `json:"callType"`
	CalleeName string `json:"calleeName"`
	CalleeNumber string `json:"calleeNumber"`
	CallerName string `json:"callerName"`
	CallerNumber string `json:"callerNumber"`
	ConversationDuration int `json:"conversationDuration"`
	DestinationNumber string `json:"destinationNumber"`
}
/*
func New_Tele2Ats(companyID int, regionID int, login string) *Tele2Ats {
	return &Tele2Ats{Company_id: companyID, Region_id: regionID, Login: login}
}
*/

func (t *Tele2Ats) LoginForQuery() string {
	return fmt.Sprintf("company_id=%d&region_id=%d&login=%s", t.CompanyID, t.RegionID, t.Login)
}

//PUT /authorization/refresh/token
func (t *Tele2Ats) RefreshTokens() error {
	client := &http.Client{}
	req, _ := http.NewRequest("PUT", API_URL + "/authorization/refresh/token", nil)
	req.Header.Set("Authorization", t.RefreshToken)
	resp, err := client.Do(req)	
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	token_time := time.Now().Add(time.Minute * time.Duration(10) * -1) //10 minutes less
	auth_data := struct {
		AccessToken string `json:"accessToken,"`
		RefreshToken string `json:"refreshToken"`
	}{}
	if err := json.Unmarshal(body, &auth_data); err != nil {
		return err
	}
	t.AccessToken = auth_data.AccessToken
	t.RefreshToken = auth_data.RefreshToken
	t.TokenDate = token_time
	return nil
}


//Добавляет в запрос токен авториизации, если надо обновляет токены
func (t *Tele2Ats) AddAuthTokenToRequest(req *http.Request) error {
	if t.AccessToken == "" || (t.TokenDate.Add(t.AccessTokenDuration).Before(time.Now())) {
		//no token or expired
fmt.Println("no token or expired - refreshing AccessToken=", t.AccessToken)		
		if t.RefreshToken == "" || (t.TokenDate.Add(t.RefreshTokenDuration).Before(time.Now())) {
fmt.Println("no token or expired - refreshing RefreshToken=", t.RefreshToken)				
			return errors.New(ERR_REFRESH_TOKEN_EXPIRED)
		} 
		if err := t.RefreshTokens(); err != nil {
			return err
		}	
fmt.Println("Tokens are refreshed successfully")
	}
	req.Header.Set("Authorization", t.AccessToken)
	return nil
}

//ПОЛУЧЕНИЕ СПИСКА СОТРУДНИКОВ КОМПАНИИ
//GET /employees?company_id={}&login={}&password={}
func (t *Tele2Ats) GetEmployees() ([]Employee, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", API_URL + "/employees", nil)
	if err := t.AddAuthTokenToRequest(req); err != nil {
		return nil, err
	}
	resp, err := client.Do(req)	
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := checkForError(resp, body); err != nil {
		return nil, err
	}
	
	var employees []Employee
	if err := json.Unmarshal(body, &employees); err != nil {
		return nil, err
	}
	
	return employees, nil
}

//СLICK 2 CALL (ВЫЗОВ ЧЕРЕЗ АТС)
//	source – номер сотрудника;
//	destination – номер клиента.
func (t *Tele2Ats) MakeCall(source, destination string) error {
	req, _ := http.NewRequest("PUT", API_URL + "/call/outgoing?"+ fmt.Sprintf("source=%s&destination=%s", source, destination), nil)
	if err := t.AddAuthTokenToRequest(req); err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)	
	if err != nil {
		resp.Body.Close()
	}
	return err
}

//ПОЛУЧЕНИЕ СПИСКА ЗАПИСЕЙ РАЗГОВОРОВ
//ПОЛУЧЕНИЕ ЗАПИСЕЙ РАЗГОВОРОВ ПО ВРЕМЕНИ
func (t *Tele2Ats) GetRecordList(start time.Time, end time.Time, caller string, callee string) ([]CallRecord, error) {
	client := &http.Client{}
	req_s := API_URL + "/call-records/info?" +
		fmt.Sprintf("start=%d&end=%d",
			start.UnixNano(),
			end.UnixNano())
	if caller != "" {
		req_s+= fmt.Sprintf("&caller=%s", caller)
	}
	if callee != "" {
		req_s+= fmt.Sprintf("&callee=%s", callee)
	}
	
	req, _ := http.NewRequest("GET", req_s, nil)		
	if err := t.AddAuthTokenToRequest(req); err != nil {
		return nil, err
	}		
	resp, err := client.Do(req)	
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := checkForError(resp, body); err != nil {
		return nil, err
	}

	var records []CallRecord
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, err
	}

	return records, nil
}

//ПОЛУЧЕНИЕ ЗАПИСИ РАЗГОВОРА ПО ID
func (t *Tele2Ats) GetRecord(filename string) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", API_URL + "/call-records/file?" + fmt.Sprintf("filename=%s",filename), nil)
	if err := t.AddAuthTokenToRequest(req); err != nil {
		return nil, err
	}		
	resp, err := client.Do(req)	
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := checkForError(resp, body); err != nil {
		return nil, err
	}
	
	return body, nil
}

//ПОЛУЧЕНИЕ ИНФОРМАЦИИ О ТЕКУЩИХ ЗВОНКАХ
//Вечный цикл ожидания новых звонков
//Возвращает только новые звонки с установленным CallAction start|end
func (t *Tele2Ats) WaitForNewCalls(pause time.Duration) <-chan *ActiveCalls {
	act_calls := make(chan *ActiveCalls)
	req, err := http.NewRequest("GET", API_URL + "/monitoring/calls", nil)
		
	var body []byte
	var resp *http.Response
	go func() {
		if err != nil {
			c := ActiveCalls{}
			c.Error = err
			act_calls <- &c
			
		}else{
			var old_crc uint32
			old_calls := make(ActiveCallList)
			for  {
				c := ActiveCalls{}
				
				old_token_date := t.TokenDate
				
				if err := t.AddAuthTokenToRequest(req); err != nil {
					c.Error	= err
					
				}else{
					if old_token_date != t.TokenDate {
						//new token is generated - notify client
						c.AccessToken = t.AccessToken
						c.RefreshToken = t.RefreshToken
						c.TokenDate = t.TokenDate
						old_token_date = t.TokenDate
					}
					client := &http.Client{}		
					resp, c.Error = client.Do(req)	
					if c.Error == nil {			
						body, c.Error = ioutil.ReadAll(resp.Body)
						if c.Error == nil && len(body)>0 {
							cur_crc := crc32.Checksum(body, crc32.IEEETable)
							if cur_crc != old_crc {
								var new_calls []ActiveCallInf
								err := json.Unmarshal(body, &new_calls)
								if err == nil {
									/*if len(new_calls) > 0 {
										fmt.Println("=== Got new calls with new CRC32", string(body))
									}*/
									for new_call_ind, new_call :=  range new_calls {
										new_call_crc := crc32.Checksum([]byte(new_call.CallType+new_call.CallerNumberFull+new_call.CalledNumberFull), crc32.IEEETable)
										new_calls[new_call_ind].Crc = new_call_crc
										if _, ok := old_calls[new_call_crc]; !ok {
											//call start
											if c.Calls == nil {
												c.Calls = make([]ActiveCall,0)
											}
											old_calls[new_call_crc] = ActiveCall{CallInf: CallInf{CallType: new_call.CallType,
													CalledNumberFull: new_call.CalledNumberFull,
													CalledNumberShort: new_call.CalledNumberShort,
													CallerNumberFull: new_call.CallerNumberFull,
													CallerNumberShort: new_call.CallerNumberShort,
												},
												CallAction: CALL_ATS_START,
											}										
											c.Calls = append(c.Calls, old_calls[new_call_crc])
										}
									}								
									for old_call_crc, old_call :=  range old_calls {
										crc_exists := false
										for _,new_call := range new_calls {
											if new_call.Crc == old_call_crc {
												crc_exists = true
												break
											}
										}
										if !crc_exists {
											//old call has ended
											if c.Calls == nil {
												c.Calls = make([]ActiveCall, 0)
											}
											c.Calls = append(c.Calls, ActiveCall{CallInf: CallInf{CallType: old_call.CallType,
													CalledNumberFull: old_call.CalledNumberFull,
													CalledNumberShort: old_call.CalledNumberShort,
													CallerNumberFull: old_call.CallerNumberFull,
													CallerNumberShort: old_call.CallerNumberShort,
												},
												CallAction: CALL_ATS_END,
											})
											delete(old_calls, old_call_crc)
										}
									}
								}else{
									//Unmarshal error
									c.Error	= errors.New(fmt.Sprintf("json.Unmarshal error:%v on data:%s", err, string(body)))
								}
								old_crc = cur_crc
							}
						}
						resp.Body.Close()
					}
				}
				if c.Error != nil || c.Calls != nil {
					act_calls <- &c
				}
				time.Sleep(pause)
			}		
		}
		close(act_calls)
	}()
	return act_calls		
}

//ПОЛУЧЕНИЕ ИНФОРМАЦИИ О ТЕКУЩИХ ЗВОНКАХ
func (t *Tele2Ats) GetActiveCalls() ([]CallInf, error) {
	req, err := http.NewRequest("GET", API_URL + "/monitoring/calls", nil)
	if err != nil {
		return nil , err
	}
	if err := t.AddAuthTokenToRequest(req); err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)	
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := checkForError(resp, body); err != nil {
		return nil, err
	}
//fmt.Println("RESPONSE:", string(body))	
	var calls []CallInf
	if err := json.Unmarshal(body, &calls); err != nil {
		return nil, err
	}
	return calls, nil	
}

func checkForError(resp *http.Response, body []byte) error{
	if resp.StatusCode != 200 {
		err_fields := struct {
			Timestamp int64 `json:"timestamp"`
			Status int `json:"status"`
			Error string `json:"error"`
			Path string `json:"path"`
		}{}
	
		if err := json.Unmarshal(body, &err_fields); err != nil {
			return err
		}
		return errors.New(fmt.Sprintf("HTTP error: %d, text: %s, path: %s", err_fields.Status, err_fields.Error, err_fields.Path))
	}
	return nil
}


