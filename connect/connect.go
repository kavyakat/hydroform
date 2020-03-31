package connect

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/kyma-incubator/hydroform/connect/types"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func (c *KymaConnector) Connect(configurationUrl string) (connector *KymaConnector, err error) {

	csrInfo, err := getCsrInfo(configurationUrl)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	csr, privateKey, err := getCertSigningRequest(csrInfo.Certificate.Subject)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	clientCert, err := getClientCert(csrInfo.CSRUrl, csr)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	c.CsrInfo = csrInfo
	c.Ca = &types.ClientCertificate{
		PrivateKey: privateKey,
		PublicKey:  clientCert,
		Csr:        csr,
	}

	err = writeClientCertificateToFile(*c.Ca)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	return c, err //returning client certificate
}

func getCsrInfo(configurationUrl string) (*types.CSRInfo, error) {
	url, err := url.Parse(configurationUrl)

	if err != nil {
		return nil, fmt.Errorf("invalid URL")
	}

	resp, err := http.Get(url.String())

	if err != nil {
		return nil, fmt.Errorf("error trying to get CSR Information : '%s'", err.Error())
	}
	if resp == nil {
		return nil, fmt.Errorf("did not recieve a response from configuration URL : '%s'", url)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recieved non OK status code from configuration URL : '%s'", url)
	}

	//unmarshal response json and store in csrInfo
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error trying to parse JSON : '%s'", err.Error())
	}

	csrInfo := types.CSRInfo{}
	err = json.Unmarshal(response, &csrInfo)
	if err != nil {
		return nil, fmt.Errorf("error trying to get CSR Information : '%s'", err.Error())
	}
	return &csrInfo, err
}

func getCertSigningRequest(subject string) (string, string, error) {
	keys, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf(err.Error())
	}

	parts := strings.Split(subject, ",")

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
		Province:           []string{"Waldorf"}, // KAVYA - gives error if empty string provided / string not provided, should be returned in subject field ideally with other data?
	}

	// create CSR
	var csrTemplate = x509.CertificateRequest{
		Subject:            pkixName,
		SignatureAlgorithm: x509.SHA256WithRSA, // KAVYA - add extensions
	}

	csrCertificate, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, keys)
	if err != nil {
		return "", "", fmt.Errorf(err.Error())
	}

	csr := pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE REQUEST", Bytes: csrCertificate,
	})

	var privateKey bytes.Buffer
	err = pem.Encode(&privateKey, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(keys)})

	if err != nil {
		return "", "", fmt.Errorf(err.Error())
	}

	return string(csr), privateKey.String(), err
}

func getClientCert(csrUrl string, csr string) (string, error) {

	// encode CSR to base64
	encodedCsr := base64.StdEncoding.EncodeToString([]byte(csr))
	type Payload struct {
		Csr string `json:"csr"`
	}

	data := Payload{
		Csr: encodedCsr,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", csrUrl, body)
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	defer resp.Body.Close()
	certificates, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	crtResponse := types.CRTResponse{}
	err = json.Unmarshal(certificates, &crtResponse)
	if err != nil {
		return "", fmt.Errorf("JSON Error")
	}
	decodedCert, err := base64.StdEncoding.DecodeString(crtResponse.ClientCRT)
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	return string(decodedCert), err
}

func (c *KymaConnector) RegisterService(apiDocs string, eventDocs string, serviceConfig string) (err error) {
	serviceDescription := new(Service)

	serviceDescription.Documentation = new(ServiceDocumentation)
	serviceDescription.Documentation.DisplayName = "Kavya's Service"
	serviceDescription.Documentation.Description = "Kavya's decsription"
	serviceDescription.Documentation.Tags = []string{"Tag0", "Tag1"}
	serviceDescription.Documentation.Type = "Kavya's Type"

	serviceDescription.Description = "Kavya's API Description"
	serviceDescription.ShortDescription = "Kavya's API Short Description"

	serviceDescription.Provider = "Kavya provider"
	serviceDescription.Name = "Kavya name"

	if serviceConfig != "" {
		log.Println("Read Service Config")
		err := c.ReadService(serviceConfig, serviceDescription)
		if err != nil {
			log.Printf("Failed to read service config: %s", serviceConfig)
			return err
		}
	}

	if apiDocs != "" {
		if serviceDescription.API == nil {
			log.Println("No Service Description")
			serviceDescription.API = new(ServiceAPI)
			serviceDescription.API.TargetURL = "http://localhost:8080/"
		}

		serviceDescription.API.Spec, err = c.getRawJsonFromDoc(apiDocs)
		if err != nil {
			return err
		}
	}

	if eventDocs != "" {
		serviceDescription.Events = new(ServiceEvent)
		serviceDescription.Events.Spec, err = c.getRawJsonFromDoc(eventDocs)
		if err != nil {
			return err
		}
	}

	jsonBytes, err := json.Marshal(serviceDescription)
	if err != nil {
		log.Printf("JSON marshal failed: %s", err)
		return
	}

	if c.CsrInfo == nil || c.CsrInfo.API.MetadataUrl == "" {
		log.Printf("%s", fmt.Errorf("metadata url is missing, cannot proceed"))
		return
	}

	client, err := c.GetSecureClient()
	resp, err := client.Post(c.CsrInfo.API.MetadataUrl, "application/json", bytes.NewBuffer(jsonBytes))

	if err != nil {
		return fmt.Errorf(err.Error())
	}

	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	if err != nil {
		log.Printf("could not dump response: %v", err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		log.Printf("Successfully registered service with")
		log.Printf(bodyString)
	} else {
		log.Printf("Status: %d >%s< \n on URL: %s", resp.StatusCode, bodyString, c.CsrInfo.API.MetadataUrl)
		return errors.New("Failed to register")
	}

	id := &struct {
		ID string `json: "id"`
	}{}

	err = json.Unmarshal(bodyBytes, id)
	if err != nil {
		log.Println("Failed to parse registration response")
		return err
	}

	log.Printf("%v", id)
	serviceDescription.id = id.ID
	serviceDescriptionString, err := json.Marshal(serviceDescription)
	ioutil.WriteFile(id.ID+".json", serviceDescriptionString, 0644)

	return err
}

func (c *KymaConnector) UpdateService(id string, apiDocs string, eventDocs string) (err error) {
	serviceDescription := new(Service)
	err = c.ReadService(id+".json", serviceDescription)
	if err != nil {
		log.Printf("Failed to read service config: %s", id+".json")
		return err
	}

	if apiDocs != "" {
		if serviceDescription.API == nil {
			serviceDescription.API = new(ServiceAPI)
			serviceDescription.API.TargetURL = "http://localhost:8080/"
		}

		serviceDescription.API.Spec, err = c.getRawJsonFromDoc(apiDocs)
		if err != nil {
			return err
		}

	}

	if eventDocs != "" {

		serviceDescription.Events = new(ServiceEvent)

		serviceDescription.Events.Spec, err = c.getRawJsonFromDoc(eventDocs)
		if err != nil {
			return err
		}

	}

	jsonBytes, err := json.Marshal(serviceDescription)
	if err != nil {
		log.Printf("JSON marshal failed: %s", err)
		return
	}

	if c.CsrInfo == nil || c.CsrInfo.API.MetadataUrl == "" {
		log.Printf("%s", fmt.Errorf("metadata url is missing, cannot proceed"))
		return
	}

	client, err := c.GetSecureClient()
	if err != nil {
		return err
	}

	url := c.CsrInfo.API.MetadataUrl + "/" + id
	log.Println(string(jsonBytes[:]))
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Couldn't register service: %s", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("Successfully registered service with")
	} else {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		log.Printf("Status: %d >%s<\n on URL: %s", resp.StatusCode, bodyString, url)
		return errors.New("Failed to Update")
	}

	return
}

func (c *KymaConnector) DeleteService(id string) (err error) {
	client, err := c.GetSecureClient()
	if err != nil {
		return err
	}

	url := c.CsrInfo.API.MetadataUrl + "/" + id
	req, _ := http.NewRequest("DELETE", url, nil)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Couldn't delete service: %s", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		log.Printf("Successfully deleted service")
		return nil
	} else {
		return errors.New("Failed to delete")
	}
}

func writeClientCertificateToFile(cert types.ClientCertificate) error {

	//dir, _ := os.Getwd()
	if cert.Csr != "" {
		//err := ioutil.WriteFile(filepath.Join(dir,"certs","generated.csr"), []byte(cert.Csr), 0644)
		err := ioutil.WriteFile("generated.csr", []byte(cert.Csr), 0644)
		if err != nil {
			return fmt.Errorf(err.Error())
		}
	}

	if cert.PublicKey != "" {
		//err := ioutil.WriteFile(filepath.Join(dir,"certs","generated.crt"), []byte(cert.PublicKey), 0644)
		err := ioutil.WriteFile("generated.crt", []byte(cert.PublicKey), 0644)
		if err != nil {
			return fmt.Errorf(err.Error())
		}
	}

	if cert.PrivateKey != "" {
		//err := ioutil.WriteFile(filepath.Join(dir,"certs","generated.key"), []byte(cert.PrivateKey), 0644)
		err := ioutil.WriteFile("generated.key", []byte(cert.PrivateKey), 0644)
		if err != nil {
			return fmt.Errorf(err.Error())
		}
	}
	return nil
}

// GetSecureClient is returning an http client with client certificate enabled
func (c *KymaConnector) GetSecureClient() (*http.Client, error) {
	cert, err := tls.X509KeyPair([]byte(c.Ca.PublicKey), []byte(c.Ca.PrivateKey))
	if err != nil {
		log.Println("Can't load certificates")
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return &http.Client{Transport: transport}, nil

}

// ReadService is loading a service description from disk
func (c *KymaConnector) ReadService(path string, s *Service) error {
	_, err := os.Stat(path)
	if err != nil {
		log.Println("No service config available")
		return err
	}

	b, err := ioutil.ReadFile(path)
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
	bytes, err := ioutil.ReadFile(doc)
	if err != nil {
		log.Println("Read error on API Docs")
		return
	}
	m = json.RawMessage(string(bytes[:]))
	return
}

func (c *KymaConnector) AddEvent(eventDoc string) (err error) {

	file, err := ioutil.ReadFile(eventDoc)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	//	event := types.Event{}

	/*err = json.Unmarshal([]byte(file), &event)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	jsonBytes, err := json.Marshal(event)
	*/
	client, err := c.GetSecureClient()
	resp, err := client.Post(c.CsrInfo.API.EventsUrl, "application/json", bytes.NewBuffer([]byte(file)))

	if err != nil {
		return fmt.Errorf(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("Successfully registered event")
		//return nil
	} else {
		log.Print("Incorrect response")
		return err
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	log.Print(string(bodyBytes))
	if err != nil {
		log.Println("Failed to parse registration response")
		return err
	}

	return err
}

func (c *KymaConnector) GetSubscribedEvents() (err error) {
	client, err := c.GetSecureClient()

	resp, err := client.Get(c.CsrInfo.API.EventsInfoUrl)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(err.Error())
	}

	//unmarshal response json and store in csrInfo
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	log.Print(string(response))
	return err
}
