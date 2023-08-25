package authorize

import (
	"context"
	"errors"
	"fmt"
	"os"

	"strconv"
	"strings"

	"time"

	//"github.com/dgrijalva/jwt-go"
	//uuid "github.com/aidarkhanov/nanoid/v2"
	//"github.com/davecgh/go-spew/spew"
	//log "github.com/sirupsen/logrus"
	"github.com/dhf0820/uc_core/common"
	//"github.com/google/uuid"
	"github.com/davecgh/go-spew/spew"
	"github.com/dhf0820/VsMongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//	type Session struct {
//		Token     string `json:"token"`
//		CacheName string `json:"cacheName"`
//	}
//
// A limited information of the patient.
type PatientSummary struct {
	ID         primitive.ObjectID `json:"id" bson:"id"`
	FullName   string             `json:"fullName" bson:"fullName"`
	LastAccess time.Time          `json:"lastAccess" bson:"lastAccess"`
}

// type SessionHistory struct {
// 	FacilityId		primitive.ObjectID	`json:"facilityId" bson:"facilityId"`
// 	SystemId		primitive.ObjectID	`json:"systemId" bson:"systemId"`
// 	//Last X patients the user has selected
// 	PatientHistory	[]PatientSummary	`json:"patientHistory" bson:"patientHistory"`
// 	Token 			string				`json:"token" bson:"token"`
// }

// SessionConnection is a remote EMR The User may connect to
// Would need to include the Facility/System Name
type SessionConnection struct {
	UserId     primitive.ObjectID `json:"userId" bson:"userId"`
	FacilityId primitive.ObjectID `json:"facilityId" bson:"facilityId"`
	SystemId   primitive.ObjectID `json:"systemId" bson:"systemId"`
	//Last X patients the user has selected
	PatientHistory []PatientSummary `json:"patientHistory" bson:"patientHistory"`
	Token          string           `json:"token" bson:"token"` //for this connection to Remote

}

// A user logged into UC receives an AuthSession
type AuthSession struct {
	ID             primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Status         int                `json:"status" bson:"status"`
	UserID         primitive.ObjectID `json:"user_id" bson:"user_id"`
	UserName       string             `json:"user_name" bson:"user_name"`
	FullName       string             `json:"fullName" bson:"fullName"`
	JWToken        string             `json:"jwToken" bson:"jwToken"`
	CurrentPatId   string             `json:"current_pat_id" bson:"current_pat_id"` //Keeps the current patient. If changes, start a new session, Delete old
	ExpiresAt      *time.Time         `json:"expiresAt" bson:"expiresAt"`
	CreatedAt      *time.Time         `json:"createdAt" bson:"createdAt"`
	LastAccessedAt *time.Time         `json:"latAccessedAt" bson:"lastAccessedAt"`
	// May not want to include this in what gets returned to the user on login
	Connections []SessionConnection `json:"connections" bson:"connections"`
}

// type Status struct {
// 	Diagnostic string `json:"diag" bson:"diag"`
// 	Reference  string `json:"ref" bson:"ref"`
// 	Patient    string `json:"pat" bson:"pat"`
// 	Encounter  string `json:"enc" bson:"enc"`
// }

func ValidateSession(id string) (*AuthSession, error) {
	id = strings.Trim(id, " ")
	if id == "" {
		VLog("ERROR", "session is blank")
		return nil, fmt.Errorf("401|Unauthorized")
	}
	VLog("INFO", "Validate Id: "+id)
	ID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New(VLogErr("ID " + id + " FromHex Error: " + err.Error()))
	}
	// Session ID is valid formay, Retrieve it, Update it and return it.
	filter := bson.M{"_id": ID}
	collection, _ := GetCollection("Sessions")
	as := &AuthSession{}
	err = collection.FindOne(context.TODO(), filter).Decode(as)
	if err != nil {
		VLog("ERROR", fmt.Sprintf("Session for ID [%s] returned ERROR: %s", id, err.Error()))
		return nil, errors.New(VLogErr("AuthSession does not exist"))
	}
	//Sessions is valid update it including new token
	err = as.UpdateSession()
	if err != nil {
		return nil, err
	}
	return as, err
}

func (as *AuthSession) UpdateTimes() error {
	tnow := time.Now().UTC()

	//log.Infof("ValidateSession:70 - Time now: %d  expireTime: %d", tnow, as.ExpireAt)
	if tnow.Unix() > as.ExpiresAt.Unix() {
		return errors.New(VLogErr("Session Expired"))
	}
	sessionLengthStr := os.Getenv("SESSION_LENGTH")
	sessionLength, err := strconv.Atoi(sessionLengthStr)
	if err != nil {
		return errors.New(VLogErr(fmt.Sprintf("Can not convert SESSION_LENGTH: [%s] to integer minutes", sessionLengthStr)))
	}
	expires := tnow.Add(time.Duration(sessionLength) * time.Minute)
	as.ExpiresAt = &expires
	return nil
}

//func (as *AuthSession) NewSessionID() error {
// 	id, err := uuid.New()
// 	if err!= nil {
// 		return fmt.Errorf("Cound not generate uuid: %s\n", err.Error())
// 	}
// 	err = DeleteAllCasheForSession(as.SessionID)
// 	as.SessionID = id
// 	filter := bson.M{"_id": as.ID}

// 	update := bson.M{"$set": bson.M{"session_id": as.SessionID}}
// 	collection, _ := storage.GetCollection("sessions")
// 	_, err = collection.UpdateOne(context.TODO(), filter, update)
// 	if err != nil {
// 		msg := fmt.Sprintf("Update SessionID failed: %s", err.Error)
// 		log.Error(msg)
// 		return errors.New(msg)
// 	}

// }

func (as *AuthSession) Create() error { // SessionID is provided
	if !as.ID.IsZero() {
		return errors.New(VLogErr("AuthSession with ID: " + as.ID.Hex() + " exists"))
	}

	// id, err := uuid.New()
	// if err != nil {
	// 	return fmt.Errorf("auth_session:95 -- Could not generate uuid: %s\n", err.Error())
	// }
	//fmt.Printf("CreateSession:100 -- cheking if session exists: %s\n", spew.Sdump(as))
	// as, err = ValidateAuth(as.Token)
	// if err == nil {
	// 	log.Infof("Session already exists for %s\n", as.Token)

	// 	as.UpdateSessionID()
	// 	return nil //errors.New("Session already exsts")
	// } else {
	// 	msg := fmt.Sprintf("auth_session:77 -- err: %s", err.Error())
	// 	log.Error(msg)
	// 	return errors.New(msg)
	// }
	// if as == nil {
	// 	log.Errorf("auth_session:76 -- as is nil returned from")
	// }
	sessionLengthStr := os.Getenv("SESSION_LENGTH")
	sessionLength, err := strconv.Atoi(sessionLengthStr)
	if err != nil {
		return errors.New(VLogErr("Can not convert SESSION_LENGTH: [" + sessionLengthStr + "] to integer minutes"))
	}
	//as.UserID = userId
	now := time.Now()
	expires := now.Add(time.Duration(sessionLength) * time.Minute)

	as.CreatedAt = &now
	as.ExpiresAt = &expires

	//as.SessionID = id
	//log.Infof("Creating Session: %s\n", spew.Sdump(as))
	err = as.Insert()
	if err != nil {
		return errors.New(VLogErr("Insert Failed err: " + err.Error()))
	}
	// filter := bson.D{{"token", as.Token}}
	// collection, _ := storage.GetCollection("sessions")

	// err = collection.FindOne(context.TODO(), filter).Decode(&as)
	// if err != nil {
	// 	fmt.Printf("Create:82 - FindFilter: %s - Err:%s\n", as.Token, err.Error())
	// }
	//fmt.Printf("Right after Insert: %s\n", spew.Sdump(as))
	return nil
}

// func (as *AuthSession) Delete() error {
// 	//startTime := time.Now()
// 	collection, _ := GetCollection("sessions")
// 	filter := bson.D{{"sessionid", as.SessionID}}
// 	//log.Debugf("    bson filter delete: %v\n", filter)
// 	_, err := collection.DeleteMany(context.Background(), filter)
// 	if err != nil {
// 		log.Errorf("!     137 -- DeleteSession for Dession %s failed: %v", as.SessionID, err)
// 		return err
// 	}
// 	//log.Infof("@@@!!!   140 -- Deleted %d Sessions for session: %v in %s", deleteResult.DeletedCount, as.SessionID, time.Since(startTime))
// 	return nil
// }

func CreateSessionForUser(user *User) (*AuthSession, error) {
	userID := user.ID

	filter := bson.M{"user_id": userID}
	collection, _ := GetCollection("sessions")
	as := &AuthSession{}
	err := collection.FindOne(context.TODO(), filter).Decode(as) // See if the user already has a session
	if err == nil {                                              // The user has a session, keep using it
		as.UpdateSession() // Extend the current session
		return as, nil
	}
	// Create a new Session
	as.UserID = userID
	as.UserName = user.UserName
	err = as.Insert()
	if err != nil {
		msg := fmt.Sprintf("insert Session error: %s", err.Error())

		return nil, errors.New(VLogErr(msg))
	}
	return as, nil
}

func GetSessionForUserID(userID primitive.ObjectID) (*common.AuthSession, error) {
	filter := bson.M{"user_id": userID}
	collection, _ := GetCollection("sessions")
	as := &common.AuthSession{}
	err := collection.FindOne(context.TODO(), filter).Decode(as) // See if the user already has a session
	return as, err
}

func ValidateAuth(id string) (*AuthSession, error) {
	//TODO: Actually validate the session as a valid JWT. Right now just using
	VLog("INFO", fmt.Sprintf("Token: [%s]", id))
	if strings.Trim(id, " ") == "" {
		return nil, errors.New(VLogErr("401|Unauthorized Token is Blank"))
	}
	oId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		msg := "ValidateAuth:199 -- Invalid SessionID"
		return nil, errors.New(VLogErr(msg))
	}
	filter := bson.M{"_id": oId}

	collection, _ := GetCollection("sessions")
	var as AuthSession
	err = collection.FindOne(context.TODO(), filter).Decode(&as)
	if err != nil {
		VLog("ERROR", fmt.Sprintf("Find Session for token %s returned ERROR: %s", id, err.Error()))
		//fmt.Printf("as: %s\n", spew.Sdump(as))
		return nil, fmt.Errorf("not Authorized")
	}
	tnow := time.Now().UTC().Unix()
	if tnow > as.ExpiresAt.Unix() {
		return nil, errors.New("notLoggedIn")
	}
	//log.Debugf("auth_session210 -- Validate Found Session: %s", spew.Sdump(as))
	//log.Debugf("Found Session: %s update ExpireAt: %s  \n\n", as.ID.String(), as.ExpireAt.String())
	//spew.Dump(as)
	as.UpdateSession()
	return &as, nil
}

func (as *AuthSession) Insert() error {
	as.ExpiresAt = as.CalculateExpireTime()
	tn := time.Now().UTC()
	as.LastAccessedAt = &tn
	collection, _ := GetCollection("Sessions")
	insertResult, err := collection.InsertOne(context.TODO(), as)
	if err != nil {
		return errors.New(VLogErr("InsertOne error: " + err.Error()))
	}
	as.ID = insertResult.InsertedID.(primitive.ObjectID)
	VLog("INFO", ("New ID: %s" + as.ID.Hex()))
	return nil
}

func (as *AuthSession) UpdateSession() error {
	saveAs := *as
	filter := bson.M{"_id": as.ID}
	as.ExpiresAt = as.CalculateExpireTime()
	tn := time.Now().UTC()
	as.LastAccessedAt = &tn
	update := bson.M{"$set": bson.M{"expire_at": as.ExpiresAt, "accessed_at": as.LastAccessedAt}}

	collection, err := GetCollection("Sessions")
	updResults, err := collection.UpdateOne(context.TODO(), filter, update)
	VLog("INFO", "updResult: "+spew.Sdump(updResults))
	if err != nil {
		*as = saveAs
		return errors.New("UpdateSession failed: " + err.Error())
	}
	return nil
}

// func (as *AuthSession) Update(update bson.M) (error) {
// 	saveAS := *as
// 	//fmt.Printf("AuthSession.Update: 274 -- as: %s\n", spew.Sdump(as))
// 	collection, _ := GetCollection("Sessions")
// 	//fmt.Printf("LIne 258\n")
// 	filter := bson.M{"_id": as.ID}
// 	//fmt.Printf("Filter: %v\n", filter)
// 	_, err := collection.UpdateOne(context.TODO(), filter, update)
// 	if err != nil {
// 		log.Errorf("AuthSession.Update:272 error %s", err)
// 		return errors.New(VLogErr("Update error: "+ err.Error()))
// 	}
// 	//log.Debugf("AuthSession.Update:275 -- Matched: %d  -- modified: %d for ID: %s", res.MatchedCount, res.ModifiedCount, as.ID.String())

// 	asUpd, err := GetSessionForUserID(as.UserID)
// 	as = asUpd
// 	return asUpd, err
// }

// func (as *AuthSession) UpdateDiagStatus(status string) (*AuthSession, error) {
// 	fmt.Printf("AuthSession.UpdateDiagStatus:292\n")
// 	as.Status.Diagnostic = status
// 	update := bson.M{"$set": bson.M{"status": as.Status}}
// 	asUpd, err := as.Update(update)
// 	if err != nil {
// 		err = fmt.Errorf("UpdateStatus:294 -- error: %s", err.Error())
// 		log.Error(err.Error())
// 		return nil, err
// 	}
// 	return asUpd, nil
// }

// func (as *AuthSession) UpdatePatStatus(status string) (*AuthSession, error) {
// 	fmt.Printf("AuthSession.UpdatePatStatus:302")
// 	as.Status.Patient = status
// 	update := bson.M{"$set": bson.M{"status": as.Status}}
// 	asUpd, err := as.Update(update)
// 	if err != nil {
// 		err = fmt.Errorf("UpdateStatus:294 -- error: %s", err.Error())
// 		log.Error(err.Error())
// 		return nil, err
// 	}
// 	return asUpd, nil
// }

// func (as *AuthSession) UpdateRefStatus(status string) (*AuthSession, error) {
// 	//fmt.Printf("AuthSession.UpdateStatus:316")
// 	as.Status.Reference = status
// 	update := bson.M{"$set": bson.M{"status": as.Status}}
// 	asUpd, err := as.Update(update)
// 	if err != nil {
// 		err = fmt.Errorf("UpdateStatus:322 -- error: %s", err.Error())
// 		log.Error(err.Error())
// 		return nil, err
// 	}
// 	return asUpd, nil
// }

// func (as *AuthSession) UpdateEncStatus(status string) (*AuthSession, error) {
// 	fmt.Printf("AuthSession.UpdateEncStatus:329")

// 	as.Status.Encounter = status
// 	update := bson.M{"$set": bson.M{"status": as.Status}}
// 	asUpd, err := as.Update(update)
// 	if err != nil {
// 		err = fmt.Errorf("UpdateStatus:335 -- error: %s", err.Error())
// 		log.Error(err.Error())
// 		return nil, err
// 	}
// 	return asUpd, nil
// }

// func (as *AuthSession) UpdateEncSessionId() (*AuthSession, error) {
// 	fmt.Printf("AuthSession.UpEncSessionId:348 --Entry: %s\n", spew.Sdump(as))

// 	id, err := uuid.New()
// 	if err != nil {
// 		return nil, fmt.Errorf("AuthSession.UpdateEncSessionId:352 -- Could not generate Enc uuid: %s", err.Error())
// 	}
// 	update := bson.M{"$set": bson.M{"enc_session_id": id}}
// 	if err != nil {
// 		return nil, fmt.Errorf("AuthSession.UpdatEncSessionId:291 -- Cound not set EncSessionID uuid: %s", err.Error())
// 	}
// 	fmt.Printf("AuthSession.UpdatEncSessionId:293 -- %s\n", spew.Sdump(as))
// 	asUpd, err := as.Update(update)
// 	as = asUpd
// 	return asUpd, err
// }

// func (as *AuthSession) UpdatePatSessionId() (*AuthSession, error) {
// 	fmt.Printf("AuthSession.UpdatePatSessionId:366 --Entry: %s\n", spew.Sdump(as))

// 	id, err := uuid.New()
// 	if err != nil {
// 		return nil, fmt.Errorf("AuthSession.UpdatePatSessionId:287 -- Cound not generate Pat uuid: %s", err.Error())
// 	}
// 	update := bson.M{"$set": bson.M{"pat_session_id": id}}
// 	if err != nil {
// 		return nil, fmt.Errorf("AuthSession.UpdatePatSessionId:291 -- Cound not set PatSessionID uuid: %s", err.Error())
// 	}
// 	fmt.Printf("AuthSession.UpdatePatSessionId:293 -- %s\n", spew.Sdump(as))
// 	asUpd, err := as.Update(update)
// 	as = asUpd
// 	return asUpd, err
// }

// func (as *AuthSession) UpdateDocSessionId() (*AuthSession, error) {
// 	fmt.Printf("AuthSession.UpdateDocSessionId:383 --Entry: %s\n", spew.Sdump(as))
// 	id, err := uuid.New()
// 	if err != nil {
// 		return nil, fmt.Errorf("auth_session.UpdateDocId:302 -- Cound not generate Doc uuid: %s", err.Error())
// 	}
// 	update := bson.M{"$set": bson.M{"doc_session_id": id}}
// 	if err != nil {
// 		return nil, fmt.Errorf("AuthSession.UpdateDocSessionId:306 -- Cound not set DocSessionID uuid: %s", err.Error())
// 	}
// 	return as.Update(update)
// }

// func (as *AuthSession) UpdateSessionID() (*AuthSession, error) {
// 	id, err := uuid.New()
// 	if err != nil {
// 		return nil, fmt.Errorf("AuthSession.UpdateSessionId:314 -- Cound not generate uuid: %s", err.Error())
// 	}
// 	update := bson.M{"$set": bson.M{"session_id": id}}
// 	return as.Update(update)

// 	// collection, _ := storage.GetCollection("sessions")
// 	// res, err := collection.UpdateOne(context.TODO(), filter, update)
// 	// if err != nil {
// 	// 	log.Errorf(" Update error %s", err)
// 	// 	return err
// 	// }
// 	// log.Debugf("auth_session:265 -- Matched: %d  -- modified: %d for ID: %s", res.MatchedCount, res.ModifiedCount, as.ID.String())
// 	//return nil
// }

func (as *AuthSession) CalculateExpireTime() *time.Time {
	loc, _ := time.LoadLocation("UTC")
	addlTime := time.Hour * 2
	ExpireAt := time.Now().In(loc).Add(addlTime)
	return &ExpireAt
}

// func (as *AuthSession) GetDocumentStatus() string {
// 	latest, _ := GetSessionForUserID(as.UserID)
// 	if latest.Status.Diagnostic == "filling" || latest.Status.Reference == "filling" {
// 		return "filling"
// 	}
// 	return "done"
// }

// func (as *AuthSession) GetDiagReptStatus() string {
// 	latest, _ := GetSessionForUserID(as.UserID)
// 	return latest.Status.Diagnostic
// }

// func (as *AuthSession) GeReptRefStatus() string {
// 	latest, _ := GetSessionForUserID(as.UserID)
// 	return latest.Status.Reference
// }
// func (as *AuthSession) GetPatientStatus() string {
// 	latest, _ := GetSessionForUserID(as.UserID)
// 	return latest.Status.Patient
// }

// func (as *AuthSession) GetEncounterStatus() string {
// 	latest, _ := GetSessionForUserID(as.UserID)
// 	return latest.Status.Encounter
// }