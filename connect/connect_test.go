package connect

import (
	"encoding/json"
	"github.com/kyma-incubator/hydroform/connect/types"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type MockWriter struct{}

func (m *MockWriter) WriteData(string, []byte) error {
	return nil
}

func (m *MockWriter) ReadData(filename string) ([]byte, error) {
	if filename == "testEvent" {
		eventJson := "{\"asyncapi\":\"1.0.0\",\"info\":{\"title\":\"PetStore Events\",\"version\":\"1.0.0\",\"description\":\"Description of all the events\"},\"baseTopic\":\"stage.com.some.company.system\",\"topics\":{\"petCreated.v1\":{\"subscribe\":{\"summary\":\"test event\",\"payload\":{\"type\":\"object\",\"properties\":{\"pet\":{\"type\":\"object\",\"required\":[\"id\",\"name\"],\"example\":{\"id\":\"4caad296-e0c5-491e-98ac-0ed118f9474e\",\"category\":\"mammal\",\"name\":\"doggie\"},\"properties\":{\"id\":{\"title\":\"Id\",\"description\":\"Resource identifier\",\"type\":\"string\"},\"name\":{\"title\":\"Name\",\"description\":\"Pet name\",\"type\":\"string\"},\"category\":{\"title\":\"Category\",\"description\":\"Animal category\",\"type\":\"string\"}}}}}}}}}"
		return []byte(eventJson), nil
	} else if filename == "testService.json" {
		serviceJson := "{\"provider\":\"Default provider\",\"name\":\"Default name\",\"description\":\"Default API Description\",\"shortDescription\":\"Default API Short Description\",\"events\":{\"spec\":{\"asyncapi\":\"1.0.0\",\"info\":{\"title\":\"PetStore Events\",\"version\":\"1.0.0\",\"description\":\"Description of all the events\"},\"baseTopic\":\"stage.com.some.company.system\",\"topics\":{\"petCreated.v1\":{\"subscribe\":{\"summary\":\"test event\",\"payload\":{\"type\":\"object\",\"properties\":{\"pet\":{\"type\":\"object\",\"required\":[\"id\",\"name\"],\"example\":{\"id\":\"4caad296-e0c5-491e-98ac-0ed118f9474e\",\"category\":\"mammal\",\"name\":\"doggie\"},\"properties\":{\"id\":{\"title\":\"Id\",\"description\":\"Resource identifier\",\"type\":\"string\"},\"name\":{\"title\":\"Name\",\"description\":\"Pet name\",\"type\":\"string\"},\"category\":{\"title\":\"Category\",\"description\":\"Animal category\",\"type\":\"string\"}}}}}}}}}},\"documentation\":{\"displayName\":\"Default Service\",\"description\":\"Default description\",\"type\":\"Default Type\",\"tags\":[\"Tag0\",\"Tag1\"]}}"
		return []byte(serviceJson), nil
	} else if filename == "config.json" {
		configJson := "{\"csrUrl\":\"test.com/csrUrl\",\"api\":{\"eventsInfoUrl\":\"test.com/eventsinfourl\",\"eventsUrl\":\"test.com/eventsurl\",\"metadataUrl\":\"test.com/metadataurl\",\"infoUrl\":\"test.com/infourl\",\"certificatesUrl\":\"test.com/certificatesurl\"},\"certificate\":{\"subject\":\"O=Organization,OU=OrgUnit,L=Waldorf,ST=Waldorf,C=DE,CN=testApp\",\"extensions\":\"\",\"key-algorithm\":\"rsa2048\"}}"
		return []byte(configJson), nil
	} else if filename == "info.json" {
		infoJson := "{\"clientIdentity\":{\"application\":\"testApp\"},\"urls\":{\"eventsInfoUrl\":\"test.com/eventsinfourl\",\"eventsUrl\":\"test.com/eventsurl\",\"metadataUrl\":\"test.com/metadataurl\",\"renewCertUrl\":\"test.com/renewcerturl\",\"revokeCertUrl\":\"test.com/revokecerturl\"}}"
		return []byte(infoJson), nil
	} else if filename == "generated.csr" {
		key := "testCsr"
		return []byte(key), nil
	} else if filename == "generated.crt" {
		key := "testPublicKey"
		return []byte(key), nil
	} else if filename == "generated.key" {
		key := "testPrivateKey"
		return []byte(key), nil
	} else {
		return []byte(""), nil
	}
}

func TestKymaConnector_Connect(t *testing.T) {
	sendCsrToKymaServer := sendCsrToKymaServer(t)
	defer sendCsrToKymaServer.Close()

	getInfoServer := getInfoServer(t)
	defer getInfoServer.Close()

	getCsrInfoServer := getCsrInfoServer(t, sendCsrToKymaServer.URL, getInfoServer.URL)
	defer getCsrInfoServer.Close()

	mockWriter := &MockWriter{}
	type fields struct {
		CsrInfo          *types.CSRInfo
		Ca               *types.ClientCertificate
		Info             *types.Info
		SecureClient     *http.Client
		StorageInterface WriterInterface
	}
	type args struct {
		configurationUrl string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "correct",
			fields: fields{
				CsrInfo: &types.CSRInfo{
					CSRUrl:      "",
					API:         &types.API{},
					Certificate: &types.Certificate{},
				},
				Ca: &types.ClientCertificate{},
				Info: &types.Info{
					ClientIdentity: &types.ClientIdentity{},
					URLs:           &types.URLs{},
				},
				SecureClient:     getCsrInfoServer.Client(),
				StorageInterface: mockWriter,
			},
			args:    args{configurationUrl: getCsrInfoServer.URL},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KymaConnector{
				CsrInfo:          tt.fields.CsrInfo,
				Ca:               tt.fields.Ca,
				Info:             tt.fields.Info,
				SecureClient:     tt.fields.SecureClient,
				StorageInterface: tt.fields.StorageInterface,
			}
			if err := c.Connect(tt.args.configurationUrl); (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKymaConnector_RegisterService(t *testing.T) {

	mockWriter := &MockWriter{}

	registerServiceServer := registerServiceServer(t)
	type fields struct {
		CsrInfo          *types.CSRInfo
		AppName          string
		Ca               *types.ClientCertificate
		SecureClient     *http.Client
		StorageInterface WriterInterface
	}
	type args struct {
		apiDocs       string
		eventDocs     string
		serviceConfig string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "correcy",
			fields: fields{
				CsrInfo: &types.CSRInfo{
					CSRUrl: "test.com/url",
					API: &types.API{
						MetadataUrl:     registerServiceServer.URL,
						EventsUrl:       "test.com/eventsurl",
						EventsInfoUrl:   "test.com/eventsinfourl",
						InfoUrl:         "test.com/infourl",
						CertificatesUrl: "test.com/certificatesurl",
					},
					Certificate: &types.Certificate{
						Subject:      "O=Organization,OU=OrgUnit,L=Waldorf,ST=Waldorf,C=DE,CN=testApplication",
						Extensions:   "",
						KeyAlgorithm: "rsa2048",
					},
				},
				AppName: "testApplication",
				Ca: &types.ClientCertificate{
					PrivateKey: "",
					PublicKey:  "",
					Csr:        "",
				},
				SecureClient:     registerServiceServer.Client(),
				StorageInterface: mockWriter,
			},
			args: args{
				apiDocs:       "",
				eventDocs:     "testEvent",
				serviceConfig: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KymaConnector{
				CsrInfo:          tt.fields.CsrInfo,
				Ca:               tt.fields.Ca,
				SecureClient:     tt.fields.SecureClient,
				StorageInterface: tt.fields.StorageInterface,
			}
			if err := c.RegisterService(tt.args.apiDocs, tt.args.eventDocs, tt.args.serviceConfig); (err != nil) != tt.wantErr {
				t.Errorf("RegisterService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKymaConnector_UpdateService(t *testing.T) {

	mockWriter := &MockWriter{}
	updateServiceServer := updateServiceServer(t)
	type fields struct {
		CsrInfo          *types.CSRInfo
		AppName          string
		Ca               *types.ClientCertificate
		SecureClient     *http.Client
		StorageInterface WriterInterface
	}
	type args struct {
		id        string
		apiDocs   string
		eventDocs string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "correct",
			fields: fields{
				CsrInfo: &types.CSRInfo{
					CSRUrl: "test.com/url",
					API: &types.API{
						MetadataUrl:     updateServiceServer.URL,
						EventsUrl:       "test.com/eventsurl",
						EventsInfoUrl:   "test.com/eventsinfourl",
						InfoUrl:         "test.com/infourl",
						CertificatesUrl: "test.com/certificatesurl",
					},
					Certificate: &types.Certificate{
						Subject:      "O=Organization,OU=OrgUnit,L=Waldorf,ST=Waldorf,C=DE,CN=testApplication",
						Extensions:   "",
						KeyAlgorithm: "rsa2048",
					},
				},
				AppName: "testApplication",
				Ca: &types.ClientCertificate{
					PrivateKey: "",
					PublicKey:  "",
					Csr:        "",
				},
				SecureClient:     updateServiceServer.Client(),
				StorageInterface: mockWriter,
			},
			args: args{
				id:        "testService",
				apiDocs:   "",
				eventDocs: "testEvent",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KymaConnector{
				CsrInfo:          tt.fields.CsrInfo,
				Ca:               tt.fields.Ca,
				SecureClient:     tt.fields.SecureClient,
				StorageInterface: tt.fields.StorageInterface,
			}
			if err := c.UpdateService(tt.args.id, tt.args.apiDocs, tt.args.eventDocs); (err != nil) != tt.wantErr {
				t.Errorf("UpdateService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKymaConnector_DeleteService(t *testing.T) {
	deleteServiceServer := deleteServiceServer(t)

	type fields struct {
		CsrInfo      *types.CSRInfo
		AppName      string
		Ca           *types.ClientCertificate
		SecureClient *http.Client
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "correct",
			fields: fields{
				CsrInfo: &types.CSRInfo{
					CSRUrl: "test.com/url",
					API: &types.API{
						MetadataUrl:     deleteServiceServer.URL,
						EventsUrl:       "test.com/eventsurl",
						EventsInfoUrl:   "test.com/eventsinfourl",
						InfoUrl:         "test.com/infourl",
						CertificatesUrl: "test.com/certificatesurl",
					},
					Certificate: &types.Certificate{
						Subject:      "O=Organization,OU=OrgUnit,L=Waldorf,ST=Waldorf,C=DE,CN=testApplication",
						Extensions:   "",
						KeyAlgorithm: "rsa2048",
					},
				},
				AppName: "testApplication",
				Ca: &types.ClientCertificate{
					PrivateKey: "",
					PublicKey:  "",
					Csr:        "",
				},
				SecureClient: deleteServiceServer.Client(),
			},
			args:    args{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KymaConnector{
				CsrInfo:      tt.fields.CsrInfo,
				Ca:           tt.fields.Ca,
				SecureClient: tt.fields.SecureClient,
			}
			if err := c.DeleteService(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("DeleteService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// should we delete the service file / db record as well here (add to writerinterface)
}

func TestKymaConnector_GetSecureClient(t *testing.T) {
	type fields struct {
		Ca *types.ClientCertificate
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "correct",
			fields: fields{Ca: &types.ClientCertificate{
				PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\n" +
					"MIIEpAIBAAKCAQEAv9MLDYsYDRI8TgtAOwiYm+4IXdtYXTqXcZfnl5gLQeRCAOEp\n" +
					"rLE9iosyt3XMbCkY3re/AcGZLkVGP6i61UjwFxGxv1aRSkafLmDITLyf7apEr/Aa\n" +
					"bUhwPYYt8oisos66Ndav5RPHHiItW80Dvf1CwmNZITjutRwLTh8KttW4J+JqXZ51\n" +
					"Fl0zE8xKj1sWgaDI9k4DOO2pp1MQDawqboy6gvuZ4+yHNM4nKbBFSQ3UwqWZMHM0\n" +
					"WSRwWvakFq6Fb1yV0bOX9T7JKYcQDtQEMMwGleblKTc16YBk58zkHlQYEjFLiIpl\n" +
					"r5YxAXTMulUp8H6mr75fpjT4FXXvgyq1gYt6UQIDAQABAoIBAHc61zDo1t8xCXi8\n" +
					"94R5+FlbX6nu74KrK3y4nYOVRtIC7Z+cVIn5dLYLhU+REanc9Y9hiICv8+VVu69P\n" +
					"0ilF961vGxtB1HblZIWwNG+2AnX4Ek+FHvf0QYeMQjzxBNUBR661LYlmfKpXNfhM\n" +
					"etn5dChdFgZXW9AIiWJaWw9/0cI/ng43ZCEIfrpkAhDuAptwj4Hp0mWSWiB8nv3g\n" +
					"RXQRG6hxZtgoOuVjqtMVmQcOGeT9Ve5od/Fl3Fr5FgF460v87oZsgHEeUBQxXIGy\n" +
					"APt+kfYljzpyyF5v9OJZv87Am9DUPkK7XuZ/M+ADf6r1Pi1cUL94zZxfY16QJx/K\n" +
					"ZGlQX90CgYEA7e7OYwwfCy5Ks4fz5EWwPGuwXk96MAmMZyQ5/JwI6GA9PBL2DGmC\n" +
					"MXjLbTtJ6Sc8stYgHrj1wAYnknnxvygapfcEd8cqDnpV1OnllP+3LlQsn8GZP75b\n" +
					"3DDV1stoXTFK0c1ZE1JhfMUwAGYdY98wPL+6ffeAibfhvXAVWDCqeb8CgYEAzmPu\n" +
					"dDbSPu4UeAvvC5zopLVxV7d3L1C/ivAZg8+yKInD8gB0WIIXUB0/vhnTEph+B2Rl\n" +
					"aG5xDg2r7QbLo4z4Z6DysQJa1PNR7mju2EjfA9RBFzHSS0DXU6CenYGsz+QmJlRZ\n" +
					"SY3UHWltTczBbDhkl1nRYooHQabHGI2vWLrmb+8CgYEAjKRzhOK+WuqTJ4o+ZXm4\n" +
					"Eg8J4sWSEWEjiDhGuoY1Ub7Jk4AVxwJ6/elMPhYku1gBLikaNW7ZfRdmPtQsTPVU\n" +
					"wzO/hVnKB2LS55cWqTt6uTzyX8CdaKuKOx722A/GcgfYFSoP9Dbm/0zD8ghqaQWd\n" +
					"ytr+TsWFSmLSYhsl0sp5ipsCgYAucDnFGFiyJCui3zyIJmQKO3EnRXahxM90WZXE\n" +
					"HMV/bZATMZr8Fzlbo1kmUvU1J+6jhylyF/eEK/tVN8Q2Jo/18TbqMRdy9tSmiiHD\n" +
					"tJHJcMa8i08/83T/shI+amER3cnfsfbtH+ZsP76CVOHokb/AdkswmtILKZV+ptKf\n" +
					"al5TLQKBgQDc/gzyDqgD4HcAaXnI5PApIbDbWQoek2ujDzLpskG6rlIDPmDNp2qc\n" +
					"JFGJu9jo7vSkrttIKWh9YidjqT0x5R0BTO2H5r1jqToEMmIO0yIHywpJCp2eH5fY\n" +
					"zdKfsRhnHxbGFbxcOPSOiXt4/ST8cU0zK3GyggQf+UE7lI4H3/KLqg==\n" +
					"-----END RSA PRIVATE KEY-----\n",
				PublicKey: "-----BEGIN CERTIFICATE-----\n" +
					"MIIELzCCAhegAwIBAgIBAjANBgkqhkiG9w0BAQsFADAPMQ0wCwYDVQQDEwRLeW1h\n" +
					"MB4XDTIwMDMzMTE3NDgzNloXDTIwMDcwMTE3NDgzNlowfjELMAkGA1UEBhMCREUx\n" +
					"EDAOBgNVBAgTB1dhbGRvcmYxEDAOBgNVBAcTB1dhbGRvcmYxEDAOBgNVBAkTB1dh\n" +
					"bGRvcmYxFTATBgNVBAoTDE9yZ2FuaXphdGlvbjEQMA4GA1UECxMHT3JnVW5pdDEQ\n" +
					"MA4GA1UEAxMHdGVzdGthdjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB\n" +
					"AL/TCw2LGA0SPE4LQDsImJvuCF3bWF06l3GX55eYC0HkQgDhKayxPYqLMrd1zGwp\n" +
					"GN63vwHBmS5FRj+outVI8BcRsb9WkUpGny5gyEy8n+2qRK/wGm1IcD2GLfKIrKLO\n" +
					"ujXWr+UTxx4iLVvNA739QsJjWSE47rUcC04fCrbVuCfial2edRZdMxPMSo9bFoGg\n" +
					"yPZOAzjtqadTEA2sKm6MuoL7mePshzTOJymwRUkN1MKlmTBzNFkkcFr2pBauhW9c\n" +
					"ldGzl/U+ySmHEA7UBDDMBpXm5Sk3NemAZOfM5B5UGBIxS4iKZa+WMQF0zLpVKfB+\n" +
					"pq++X6Y0+BV174MqtYGLelECAwEAAaMnMCUwDgYDVR0PAQH/BAQDAgeAMBMGA1Ud\n" +
					"JQQMMAoGCCsGAQUFBwMCMA0GCSqGSIb3DQEBCwUAA4ICAQCBaIGGX0Z1EhBTs6dG\n" +
					"CjTOc2FFRUwM8aHgxEjBKT2LTOA6hvG3BA5pflzFPkkup7K2j4EvI+NKYWyhiykJ\n" +
					"VtmblWobcq9f/RWonKE4IXYNdNaWezsts5nnW7NOvZR6JiRNalYn5HU3IVNq3oMJ\n" +
					"kdYjbWGZqz+gKNGuMYF6sQOzEjxNquz9rWXfWnoNQ5RL168zG818vpEAEC7ZrcDE\n" +
					"DYkI78WPysSd721l8yrFz7geyYshrRPJnBAOWBfrpMvfc9J+EXYwsSeIQbI5KeQn\n" +
					"peX55TYBGHt2Bf3D1sp1K9O4TWdYx+CfYm9VrpZ/yaODNi867SMi5EbqETbP9Akd\n" +
					"5IzQIttqJH2yaWPIRGkg/JXK7pBIJAyoZVYZc+WkGosN6DkAymPktRKiFbv2TnQd\n" +
					"xNre2PlOdZKRABxg345KcEIWQWdnE09lVZtgpP2YTpbHCoQMQvsqEt+CFZ4+0GhN\n" +
					"BB05i84UClBQeon6BV1av80KTBsgxftV8+qZA7TeQjwjW0slKbNFiAd+R4fGR+pf\n" +
					"9dEsmK3aYpTpcpNKpqul/IPyEa0tLEFb+Llk3wKxwzogtg7nWXK101cqmfi++6tV\n" +
					"H4f9daIj0JuPM0z0zcrPQKgkiKn4Evn81xmKfrDs+2+YHxe8igWwbFnheLlDKOOU\n" +
					"56q/TqpSCnYK7yQf2S/0KeaJ3w==\n" +
					"-----END CERTIFICATE-----\n",
				Csr: "test",
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KymaConnector{
				Ca: tt.fields.Ca,
			}
			_, err := c.GetSecureClient()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecureClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestKymaConnector_AddEvent(t *testing.T) {
	addEventServer := addEventServer(t)
	type fields struct {
		CsrInfo      *types.CSRInfo
		AppName      string
		Ca           *types.ClientCertificate
		SecureClient *http.Client
	}
	type args struct {
		event types.Event
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "correct",
			fields: fields{
				CsrInfo: &types.CSRInfo{
					CSRUrl: "test.com/csrurl",
					API: &types.API{
						MetadataUrl:     "test.com/metadataurl",
						EventsUrl:       addEventServer.URL,
						EventsInfoUrl:   "test.com/eventsinfourl",
						InfoUrl:         "test.com/infourl",
						CertificatesUrl: "test.com/certificatesurl",
					},
					Certificate: &types.Certificate{
						Subject:      "O=Organization,OU=OrgUnit,L=Waldorf,ST=Waldorf,C=DE,CN=testApplication",
						Extensions:   "",
						KeyAlgorithm: "rsa2048",
					},
				},
				AppName: "testApplication",
				Ca: &types.ClientCertificate{
					PrivateKey: "",
					PublicKey:  "",
					Csr:        "",
				},
				SecureClient: addEventServer.Client(),
			},
			args:    args{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KymaConnector{
				CsrInfo:      tt.fields.CsrInfo,
				Ca:           tt.fields.Ca,
				SecureClient: tt.fields.SecureClient,
			}
			if err := c.AddEvent(tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("AddEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKymaConnector_GetSubscribedEvents(t *testing.T) {
	eventsInfoServer := getEventsInfoServer(t)
	type fields struct {
		CsrInfo          *types.CSRInfo
		Ca               *types.ClientCertificate
		Info             *types.Info
		SecureClient     *http.Client
		StorageInterface WriterInterface
	}
	tests := []struct {
		name    string
		fields  fields
		want    []types.EventResponse
		wantErr bool
	}{
		{
			name: "correct",
			fields: fields{
				CsrInfo: &types.CSRInfo{
					CSRUrl: "test.com/url",
					API: &types.API{
						MetadataUrl:     "test.com/metadataurl",
						EventsUrl:       "test.com/eventsurl",
						EventsInfoUrl:   eventsInfoServer.URL,
						InfoUrl:         "test.com/infourl",
						CertificatesUrl: "test.com/certificatesurl",
					},
					Certificate: &types.Certificate{
						Subject:      "O=Organization,OU=OrgUnit,L=Waldorf,ST=Waldorf,C=DE,CN=testApplication",
						Extensions:   "",
						KeyAlgorithm: "rsa2048",
					},
				},
				Ca:               nil,
				Info:             nil,
				SecureClient:     eventsInfoServer.Client(),
				StorageInterface: nil,
			},
			want: []types.EventResponse{{
				Name:    "event1",
				Version: "v1",
			}, {
				Name:    "event2",
				Version: "v2",
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KymaConnector{
				CsrInfo:          tt.fields.CsrInfo,
				Ca:               tt.fields.Ca,
				Info:             tt.fields.Info,
				SecureClient:     tt.fields.SecureClient,
				StorageInterface: tt.fields.StorageInterface,
			}
			got, err := c.GetSubscribedEvents()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSubscribedEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSubscribedEvents() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetKymaConnector(t *testing.T) {
	mockWriter := &MockWriter{}
	type args struct {
		writerInterface WriterInterface
	}
	tests := []struct {
		name string
		args args
		want *KymaConnector
	}{
		{
			name: "correct",
			args: args{writerInterface: mockWriter},
			want: &KymaConnector{
				CsrInfo: &types.CSRInfo{
					CSRUrl: "test.com/csrUrl",
					API: &types.API{
						MetadataUrl:     "test.com/metadataurl",
						EventsUrl:       "test.com/eventsurl",
						EventsInfoUrl:   "test.com/eventsinfourl",
						InfoUrl:         "test.com/infourl",
						CertificatesUrl: "test.com/certificatesurl",
					},
					Certificate: &types.Certificate{
						Subject:      "O=Organization,OU=OrgUnit,L=Waldorf,ST=Waldorf,C=DE,CN=testApp",
						Extensions:   "",
						KeyAlgorithm: "rsa2048",
					},
				},
				Ca: &types.ClientCertificate{
					PrivateKey: "testPrivateKey",
					PublicKey:  "testPublicKey",
					Csr:        "testCsr",
				},
				Info: &types.Info{
					ClientIdentity: &types.ClientIdentity{AppName: "testApp"},
					URLs: &types.URLs{
						MetadataUrl:   "test.com/metadataurl",
						EventsUrl:     "test.com/eventsurl",
						RenewCertUrl:  "test.com/renewcerturl",
						RevokeCertUrl: "test.com/revokecerturl",
					},
				},
				SecureClient:     nil,
				StorageInterface: mockWriter,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetKymaConnector(tt.args.writerInterface); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetKymaConnector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKymaConnector_RenewCertificateSigningRequest(t *testing.T) {
	renewCertServer := renewCertServer(t)
	defer renewCertServer.Close()

	mockWriter := &MockWriter{}

	type fields struct {
		CsrInfo          *types.CSRInfo
		Ca               *types.ClientCertificate
		Info             *types.Info
		SecureClient     *http.Client
		StorageInterface WriterInterface
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "correct",
			fields: fields{
				CsrInfo: &types.CSRInfo{},
				Ca: &types.ClientCertificate{
					PrivateKey: "testPrivKey",
					PublicKey:  "testPubKey",
					Csr:        "testCsr",
				},
				Info: &types.Info{
					ClientIdentity: &types.ClientIdentity{AppName: "testApp"},
					URLs: &types.URLs{
						MetadataUrl:   "test.com/metadataurl",
						EventsUrl:     "test.com/eventsurl",
						RenewCertUrl:  renewCertServer.URL,
						RevokeCertUrl: "test.com/revokecerturl",
					},
				},
				SecureClient:     renewCertServer.Client(),
				StorageInterface: mockWriter,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KymaConnector{
				CsrInfo:          tt.fields.CsrInfo,
				Ca:               tt.fields.Ca,
				Info:             tt.fields.Info,
				SecureClient:     tt.fields.SecureClient,
				StorageInterface: tt.fields.StorageInterface,
			}
			if err := c.RenewCertificateSigningRequest(); (err != nil) != tt.wantErr {
				t.Errorf("RenewCertificateSigningRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKymaConnector_RevokeCertificate(t *testing.T) {
	revokeCertServer := revokeCertServer(t)
	defer revokeCertServer.Close()

	mockWriter := &MockWriter{}
	type fields struct {
		CsrInfo          *types.CSRInfo
		Ca               *types.ClientCertificate
		Info             *types.Info
		SecureClient     *http.Client
		StorageInterface WriterInterface
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "correct",
			fields: fields{
				CsrInfo: &types.CSRInfo{},
				Ca: &types.ClientCertificate{
					PrivateKey: "testPrivKey",
					PublicKey:  "testPubKey",
					Csr:        "testCsr",
				},
				Info: &types.Info{
					ClientIdentity: &types.ClientIdentity{AppName: "testApp"},
					URLs: &types.URLs{
						MetadataUrl:   "test.com/metadataurl",
						EventsUrl:     "test.com/eventsurl",
						RenewCertUrl:  "test.com/renewcerturl",
						RevokeCertUrl: revokeCertServer.URL,
					},
				},
				SecureClient:     revokeCertServer.Client(),
				StorageInterface: mockWriter,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KymaConnector{
				CsrInfo:          tt.fields.CsrInfo,
				Ca:               tt.fields.Ca,
				Info:             tt.fields.Info,
				SecureClient:     tt.fields.SecureClient,
				StorageInterface: tt.fields.StorageInterface,
			}
			if err := c.RevokeCertificate(); (err != nil) != tt.wantErr {
				t.Errorf("RevokeCertificate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func sendCsrToKymaServer(t *testing.T) *httptest.Server {
	sendCsrToKymaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got '%s'", r.Method)
		}
		reqJson, err := ioutil.ReadAll(r.Body)

		csrResponse := types.CSRResponse{}
		err = json.Unmarshal(reqJson, &csrResponse)

		if err != nil {
			t.Errorf("Unexpected error in parsing JSON ")
		}
		//fmt.Print(csrResponse)

		crtResponse := types.CRTResponse{
			CRT:       "crtEncoded",
			CaCRT:     "caCrtEncoded",
			ClientCRT: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVIekNDQWdlZ0F3SUJBZ0lCQWpBTkJna3Foa2lHOXcwQkFRc0ZBREFQTVEwd0N3WURWUVFERXdSTGVXMWgKTUI0WERUSXdNRE15TURFd01EQXdPRm9YRFRJd01EWXlNREV3TURBd09Gb3diakVMTUFrR0ExVUVCaE1DUkVVeApFREFPQmdOVkJBZ1RCMWRoYkdSdmNtWXhFREFPQmdOVkJBY1RCMWRoYkdSdmNtWXhGVEFUQmdOVkJBb1RERTl5CloyRnVhWHBoZEdsdmJqRVFNQTRHQTFVRUN4TUhUM0puVlc1cGRERVNNQkFHQTFVRUF4TUpiWGwwWlhOMFlYQncKTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE5SEUrNXRWQUR0amVIQmFPcUluNApMZi9MVytaR0dZN0RCNzAwa2RrUDdIMEphRklPSngxUkFYK2RlVklEWFgyL1d0UFZKNE0rY3d1RmpDUitPUG1lCmVtNGRHOXN1WTJvaC9xY3daZ1hJcTVtTXZ0V2FteUR4dTQ5VFhYQXlMTnFJYUN1MFQxMkQ1U2FURnhlWmY0NVUKTmtYV2kwbDBPWE5wczlxa3ZialdBdWF3eTk1bjBsOGdDRmZDRFJaSHZsOTZUSlJ6dStTSEs3bmZwMXJ3N2taMQpBbjRLa3dLd3VrOGxNVDluV3BiSGVscmpkTzhzWGExcWprQTY3RGkvNFFhck11RU1HMUJ4ZUMvR1RZUnEzN1ZCCkhwNWlFcGJqT0NzeERCcEJZWnk5MGxaVk56eTdMTytUQU44YTBPZ2g3UWhuNlJtc2hPZ3YvY05XM09YTmx6RFkKelFJREFRQUJveWN3SlRBT0JnTlZIUThCQWY4RUJBTUNCNEF3RXdZRFZSMGxCQXd3Q2dZSUt3WUJCUVVIQXdJdwpEUVlKS29aSWh2Y05BUUVMQlFBRGdnSUJBRFVBWW1OTWFvZnBWMG4vYXF0dzA3WFoxREFleXl1UjQzRUJFYk15ClhCZm16VXQ4cUsrYlJ4TDFpcEh4Y3BSOTJRUGNlcXRYRWFpcmxwSStDcXdDSXQ3ekU2b05SNEphYnBwMGlGcE4KcndreFRiR1ZrOXUrdU1xVWs4ZllnVUdKRVQyQUVLYWc3V1Q5elRyS2hmQlN4MWdCZkF2SExXaXFmU0V4T3RmWgpQSlRKeTBZM0JNdDlXTzhUMTN5dmYyd3pBVzNhSlpzU21LWUE2bndrRWcwcDVrSVYybHczNUFOblljbmxoT1c2CkcvMkp4ZXJzVVNOemhZRGZnSmhJTGsvdDdodUZNM1JKaGNmRDVZNStRRy9hZ1RGVDZRaWx3OGZyR3NTd2RXWUoKZ3dINWhGWVFOQlY3MVNzN3haLzlTNXF6SDg0T3creWxCeG9tTzc1NmhjMnoxSzhJOUFOWWJuUis2SDJYakZkMgo2cDV6Vi81cXNSSjJwblFlZlB3NTBPMVE5ektSMFVJZHcvOXlZM3NqUWY5a3F2VllDenBXT0NSVkFZcFp2dHkyCmhQUDVFdzBvN3JhcUhHT1lTYWpaNWoyaDNRRElCM3YxM0lkYUlkWm0xNkdXNFN0OXc1dURaa0VxQlZybzVOZXAKYnVGWHFMWkRYUDNMQVJXaFAzU003MS8wWWExNUROTnlPS3NDUHJ4YTUwTTlMM1dxajFwdjhjd29oVnYyT1BaaQpmNWF6THA2b0YxdFM1SkN1OFFBRjRDU0FHY25QTS9nNUFKYjN6ZTZhYXdhaFF1cThmbHhiRzRQcTYxV0w2czUyCm5uZ0kwVElsTzRnK2ZJSDlJVlF0b3I1RU1NRW5wSXpmSWM4VlJpSTN0K21FUDBJc2I3dmpNVElDclpGeFFHaEoKdk81WgotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
		}
		js, err := json.Marshal(crtResponse)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)

	}))
	return sendCsrToKymaServer
}
func getCsrInfoServer(t *testing.T, csrUrl string, infoUrl string) *httptest.Server {
	getCsrInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got '%s", r.Method)
		}

		csrInfo := types.CSRInfo{
			CSRUrl: csrUrl,
			API: &types.API{
				MetadataUrl:     "test.com/metadataurl",
				EventsUrl:       "test.com/eventsurl",
				EventsInfoUrl:   "test.com/eventsinfourl",
				InfoUrl:         infoUrl,
				CertificatesUrl: "test.com/certificatesurl",
			},
			Certificate: &types.Certificate{
				Subject:      "O=Organization,OU=OrgUnit,L=Waldorf,ST=Waldorf,C=DE,CN=testApplication",
				Extensions:   "",
				KeyAlgorithm: "rsa2048",
			},
		}
		js, err := json.Marshal(csrInfo)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}))

	return getCsrInfoServer
}
func getInfoServer(t *testing.T) *httptest.Server {
	getInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got '%s", r.Method)
		}

		info := types.Info{
			ClientIdentity: &types.ClientIdentity{AppName: "testApp"},
			URLs: &types.URLs{
				MetadataUrl:   "test.com/metadataUrl",
				EventsUrl:     "test.com/eventsUrl",
				RenewCertUrl:  "test.com/renewCertUrl",
				RevokeCertUrl: "test.com/revokeCertUrl",
			},
		}
		js, err := json.Marshal(info)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}))

	return getInfoServer
}
func registerServiceServer(t *testing.T) *httptest.Server {
	registerServiceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got '%s'", r.Method)
		}

		type serviceResponse struct {
			Id string `json:"id"`
		}
		serviceResponseJson := serviceResponse{
			Id: "testId",
		}

		js, err := json.Marshal(serviceResponseJson)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}))

	return registerServiceServer
}
func deleteServiceServer(t *testing.T) *httptest.Server {
	deleteServiceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		if r.Method != "DELETE" {
			t.Errorf("Expected 'DELETE' request, got '%s'", r.Method)
		}
	}))

	return deleteServiceServer
}
func updateServiceServer(t *testing.T) *httptest.Server {
	updateServiceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method != "PUT" {
			t.Errorf("Expected 'PUT' request, got '%s'", r.Method)
		}
	}))

	return updateServiceServer
}
func addEventServer(t *testing.T) *httptest.Server {
	addEventServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got '%s'", r.Method)
		}
		type eventResponse struct {
			eventId string
		}
		eventResponseObj := eventResponse{
			eventId: "testId",
		}
		js, err := json.Marshal(eventResponseObj)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)

	}))
	return addEventServer
}
func getEventsInfoServer(t *testing.T) *httptest.Server {
	getInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got '%s", r.Method)
		}

		jsonString := "{\"eventsInfo\":[{\"name\":\"event1\",\"version\":\"v1\"},{\"name\":\"event2\",\"version\":\"v2\"}]}"
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonString))
	}))

	return getInfoServer
}
func renewCertServer(t *testing.T) *httptest.Server {
	renewCertServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got '%s'", r.Method)
		}
		reqJson, err := ioutil.ReadAll(r.Body)

		csrResponse := types.CSRResponse{}
		err = json.Unmarshal(reqJson, &csrResponse)

		if err != nil {
			t.Errorf("Unexpected error in parsing JSON ")
		}
		//fmt.Print(csrResponse)

		crtResponse := types.CRTResponse{
			CRT:       "crtEncoded",
			CaCRT:     "caCrtEncoded",
			ClientCRT: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVIekNDQWdlZ0F3SUJBZ0lCQWpBTkJna3Foa2lHOXcwQkFRc0ZBREFQTVEwd0N3WURWUVFERXdSTGVXMWgKTUI0WERUSXdNRE15TURFd01EQXdPRm9YRFRJd01EWXlNREV3TURBd09Gb3diakVMTUFrR0ExVUVCaE1DUkVVeApFREFPQmdOVkJBZ1RCMWRoYkdSdmNtWXhFREFPQmdOVkJBY1RCMWRoYkdSdmNtWXhGVEFUQmdOVkJBb1RERTl5CloyRnVhWHBoZEdsdmJqRVFNQTRHQTFVRUN4TUhUM0puVlc1cGRERVNNQkFHQTFVRUF4TUpiWGwwWlhOMFlYQncKTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE5SEUrNXRWQUR0amVIQmFPcUluNApMZi9MVytaR0dZN0RCNzAwa2RrUDdIMEphRklPSngxUkFYK2RlVklEWFgyL1d0UFZKNE0rY3d1RmpDUitPUG1lCmVtNGRHOXN1WTJvaC9xY3daZ1hJcTVtTXZ0V2FteUR4dTQ5VFhYQXlMTnFJYUN1MFQxMkQ1U2FURnhlWmY0NVUKTmtYV2kwbDBPWE5wczlxa3ZialdBdWF3eTk1bjBsOGdDRmZDRFJaSHZsOTZUSlJ6dStTSEs3bmZwMXJ3N2taMQpBbjRLa3dLd3VrOGxNVDluV3BiSGVscmpkTzhzWGExcWprQTY3RGkvNFFhck11RU1HMUJ4ZUMvR1RZUnEzN1ZCCkhwNWlFcGJqT0NzeERCcEJZWnk5MGxaVk56eTdMTytUQU44YTBPZ2g3UWhuNlJtc2hPZ3YvY05XM09YTmx6RFkKelFJREFRQUJveWN3SlRBT0JnTlZIUThCQWY4RUJBTUNCNEF3RXdZRFZSMGxCQXd3Q2dZSUt3WUJCUVVIQXdJdwpEUVlKS29aSWh2Y05BUUVMQlFBRGdnSUJBRFVBWW1OTWFvZnBWMG4vYXF0dzA3WFoxREFleXl1UjQzRUJFYk15ClhCZm16VXQ4cUsrYlJ4TDFpcEh4Y3BSOTJRUGNlcXRYRWFpcmxwSStDcXdDSXQ3ekU2b05SNEphYnBwMGlGcE4KcndreFRiR1ZrOXUrdU1xVWs4ZllnVUdKRVQyQUVLYWc3V1Q5elRyS2hmQlN4MWdCZkF2SExXaXFmU0V4T3RmWgpQSlRKeTBZM0JNdDlXTzhUMTN5dmYyd3pBVzNhSlpzU21LWUE2bndrRWcwcDVrSVYybHczNUFOblljbmxoT1c2CkcvMkp4ZXJzVVNOemhZRGZnSmhJTGsvdDdodUZNM1JKaGNmRDVZNStRRy9hZ1RGVDZRaWx3OGZyR3NTd2RXWUoKZ3dINWhGWVFOQlY3MVNzN3haLzlTNXF6SDg0T3creWxCeG9tTzc1NmhjMnoxSzhJOUFOWWJuUis2SDJYakZkMgo2cDV6Vi81cXNSSjJwblFlZlB3NTBPMVE5ektSMFVJZHcvOXlZM3NqUWY5a3F2VllDenBXT0NSVkFZcFp2dHkyCmhQUDVFdzBvN3JhcUhHT1lTYWpaNWoyaDNRRElCM3YxM0lkYUlkWm0xNkdXNFN0OXc1dURaa0VxQlZybzVOZXAKYnVGWHFMWkRYUDNMQVJXaFAzU003MS8wWWExNUROTnlPS3NDUHJ4YTUwTTlMM1dxajFwdjhjd29oVnYyT1BaaQpmNWF6THA2b0YxdFM1SkN1OFFBRjRDU0FHY25QTS9nNUFKYjN6ZTZhYXdhaFF1cThmbHhiRzRQcTYxV0w2czUyCm5uZ0kwVElsTzRnK2ZJSDlJVlF0b3I1RU1NRW5wSXpmSWM4VlJpSTN0K21FUDBJc2I3dmpNVElDclpGeFFHaEoKdk81WgotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
		}
		js, err := json.Marshal(crtResponse)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)

	}))
	return renewCertServer
}
func revokeCertServer(t *testing.T) *httptest.Server {
	renewCertServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got '%s'", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
	}))
	return renewCertServer
}
