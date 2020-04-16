package connect

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/kyma-incubator/hydroform/connect/types"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func (c *KymaConnector) populateCsrInfo(configurationUrl string) error {
	url, _ := url.Parse(configurationUrl)

	resp, err := http.Get(url.String())

	if err != nil {
		return fmt.Errorf("error trying to get CSR Information : '%s'", err.Error())
	}
	if resp == nil {
		return fmt.Errorf("did not recieve a response from configuration URL : '%s'", url)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("recieved non OK status code from configuration URL : '%s'", url)
	}

	//unmarshal response json and store in csrInfo
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error trying to parse JSON : '%s'", err.Error())
	}

	csrInfo := types.CSRInfo{}
	err = json.Unmarshal(response, &csrInfo)
	if err != nil {
		return fmt.Errorf("error trying to get CSR Information : '%s'", err.Error())
	}

	c.CsrInfo = &csrInfo
	err = c.StorageInterface.WriteData("config.json", response)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	return err
}

func (c *KymaConnector) populateInfo() error {

	resp, err := c.SecureClient.Get(c.CsrInfo.API.InfoUrl)

	if err != nil {
		return fmt.Errorf("error trying to get info : '%s'", err.Error())
	}
	if resp == nil {
		return fmt.Errorf("did not recieve a response from info URL")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("recieved non OK status code from info URL")
	}

	//unmarshal response json and store in csrInfo
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error trying to parse JSON : '%s'", err.Error())
	}

	info := types.Info{}
	err = json.Unmarshal(response, &info)
	if err != nil {
		return fmt.Errorf("error trying to get CSR Information : '%s'", err.Error())
	}

	c.Info = &info

	err = c.StorageInterface.WriteData("info.json", response)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	return err
}

func (c *KymaConnector) populateCertSigningRequest() error {
	keys, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	parts := strings.Split(c.CsrInfo.Certificate.Subject, ",")

	var org, orgUnit, location, street, country, appName string

	for i := range parts {
		subjectTitle := strings.Split(parts[i], "=")
		switch subjectTitle[0] {
		case "O":
			org = subjectTitle[1]
		case "OU":
			orgUnit = subjectTitle[1]
		case "L":
			location = subjectTitle[1]
		case "ST":
			street = subjectTitle[1]
		case "C":
			country = subjectTitle[1]
		case "CN":
			appName = subjectTitle[1]
		}
	}

	pkixName := pkix.Name{
		Organization:       []string{org},
		OrganizationalUnit: []string{orgUnit},
		Locality:           []string{location},
		StreetAddress:      []string{street},
		Country:            []string{country},
		CommonName:         appName,
		Province:           []string{"Waldorf"},
	}

	// create CSR
	var csrTemplate = x509.CertificateRequest{
		Subject:            pkixName,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csrCertificate, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, keys)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	csr := pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE REQUEST", Bytes: csrCertificate,
	})

	var privateKey bytes.Buffer
	err = pem.Encode(&privateKey, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(keys)})

	if err != nil {
		return fmt.Errorf(err.Error())
	}

	c.Ca.PrivateKey = privateKey.String()
	c.Ca.Csr = string(csr)
	return err
}

func (c *KymaConnector) populateClientCert() error {

	// encode CSR to base64
	encodedCsr := base64.StdEncoding.EncodeToString([]byte(c.Ca.Csr))
	type Payload struct {
		Csr string `json:"csr"`
	}

	data := Payload{
		Csr: encodedCsr,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", c.CsrInfo.CSRUrl, body)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	defer resp.Body.Close()
	certificates, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	crtResponse := types.CRTResponse{}
	err = json.Unmarshal(certificates, &crtResponse)
	if err != nil {
		return fmt.Errorf("JSON Error")
	}
	decodedCert, err := base64.StdEncoding.DecodeString(crtResponse.ClientCRT)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	c.Ca.PublicKey = string(decodedCert)
	return err
}

func (c *KymaConnector) persistCertificate() error {
	if c.Ca.Csr != "" {
		err := c.StorageInterface.WriteData("generated.csr", []byte(c.Ca.Csr))
		if err != nil {
			return fmt.Errorf(err.Error())
		}
	}

	if c.Ca.PublicKey != "" {
		err := c.StorageInterface.WriteData("generated.crt", []byte(c.Ca.PublicKey))
		if err != nil {
			return fmt.Errorf(err.Error())
		}
	}

	if c.Ca.PrivateKey != "" {
		err := c.StorageInterface.WriteData("generated.key", []byte(c.Ca.PrivateKey))
		if err != nil {
			return fmt.Errorf(err.Error())
		}
	}
	return nil
}

func (c *KymaConnector) readService(path string, s *Service) error {
	path = path + ".json"

	b, err := c.StorageInterface.ReadData(path)
	if err != nil {
		log.Println("Failed to read file")
		return err
	}

	err = json.Unmarshal(b, s)
	if err != nil {
		log.Println("Failed to parse json")
		return err
	}

	return nil
}

func (c *KymaConnector) getRawJsonFromDoc(doc string) (m json.RawMessage, err error) {
	bytes, err := c.StorageInterface.ReadData(doc)
	if err != nil {
		log.Println("Read error")
		return nil, fmt.Errorf(err.Error())
	}
	m = json.RawMessage(string(bytes[:]))
	return
}

func (c *KymaConnector) populateClient() (err error) {
	c.SecureClient, err = c.GetSecureClient()
	return err
}

func (c *KymaConnector) loadConfig() error {
	config, err := c.StorageInterface.ReadData("config.json")
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	csrInfo := &types.CSRInfo{}
	json.Unmarshal(config, csrInfo)
	c.CsrInfo = csrInfo

	_, err = os.Stat("info.json")
	info, err := c.StorageInterface.ReadData("info.json")
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	infoObj := &types.Info{}
	json.Unmarshal(info, infoObj)
	c.Info = infoObj

	csr, err := c.StorageInterface.ReadData("generated.csr")
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	c.Ca.Csr = string(csr[:])

	crt, err := c.StorageInterface.ReadData("generated.crt")
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	c.Ca.PublicKey = string(crt[:])

	key, err := c.StorageInterface.ReadData("generated.key")
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	c.Ca.PrivateKey = string(key[:])

	return err
}
