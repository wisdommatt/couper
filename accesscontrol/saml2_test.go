package accesscontrol_test

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/internal/test"
)

func Test_NewSAML2ACS(t *testing.T) {
	helper := test.New(t)

	type testCase struct {
		metadataFile, acsUrl, spEntityId string
		arrayAttributes                  []string
		expErrMsg                        string
		shouldFail                       bool
	}

	for _, tc := range []testCase{
		{"testdata/idp-metadata.xml", "http://www.examle.org/saml/acs", "my-sp-entity-id", []string{}, "", false},
		{"not-there.xml", "http://www.examle.org/saml/acs", "my-sp-entity-id", []string{}, "not-there.xml: no such file or directory", true},
	} {
		metadata, err := reader.ReadFromAttrFile("saml2", "", tc.metadataFile)
		if err != nil {
			readErr := err.(errors.GoError)
			if tc.shouldFail {
				if !strings.HasSuffix(readErr.LogError(), tc.expErrMsg) {
					t.Errorf("Want: %q, got: %q", tc.expErrMsg, readErr.LogError())
				}
				continue
			}
			t.Error(err)
			continue
		}

		_, err = ac.NewSAML2ACS(metadata, "test", tc.acsUrl, tc.spEntityId, tc.arrayAttributes)
		helper.Must(err)
	}
}

func Test_SAML2ACS_Validate(t *testing.T) {
	metadata, err := reader.ReadFromAttrFile("saml2", "", "testdata/idp-metadata.xml")
	if err != nil || metadata == nil {
		t.Fatal("Expected a metadata object")
	}
	sa, err := ac.NewSAML2ACS(metadata, "test", "http://www.examle.org/saml/acs", "my-sp-entity-id", []string{"memberOf"})
	if err != nil || sa == nil {
		t.Fatal("Expected a saml acs object")
	}

	type testCase struct {
		name    string
		payload string
		wantErr bool
	}
	for _, tc := range []testCase{
		{
			"invalid body",
			"1qp4ghn1pin",
			true,
		},
		{
			"invalid SAMLResponse",
			"SAMLResponse=1qp4ghn1pin",
			true,
		},
		{
			"invalid url-encoded SAMLResponse",
			"SAMLResponse=" + url.QueryEscape("abcde"),
			true,
		},
		{
			"invalid base64- and url-encoded SAMLResponse",
			"SAMLResponse=" + url.QueryEscape(base64.StdEncoding.EncodeToString([]byte("abcde"))),
			true,
		},
		// TODO how to make test for valid SAMLResponse?
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tc.payload))
			if err := sa.Validate(req); (err != nil) != tc.wantErr {
				t.Errorf("%s: Validate() error = %v, wantErr %v", tc.name, err, tc.wantErr)
			}
		})
	}
}

func Test_SAML2ACS_ValidateAssertionInfo(t *testing.T) {
	metadata, err := reader.ReadFromAttrFile("saml2", "", "testdata/idp-metadata.xml")
	sa, err := ac.NewSAML2ACS(metadata, "test", "http://www.examle.org/saml/acs", "my-sp-entity-id", []string{"memberOf"})
	if err != nil || sa == nil {
		t.Fatal("Expected a saml acs object")
	}

	type testCase struct {
		name          string
		assertionInfo *saml2.AssertionInfo
		wantErr       bool
	}
	for _, tc := range []testCase{
		{
			"assertion mismatch",
			&saml2.AssertionInfo{
				WarningInfo: &saml2.WarningInfo{
					NotInAudience: true,
				},
			},
			true,
		},
		{
			"assertion match",
			&saml2.AssertionInfo{
				WarningInfo: &saml2.WarningInfo{},
			},
			false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err = sa.ValidateAssertionInfo(tc.assertionInfo); (err != nil) != tc.wantErr {
				t.Errorf("%s: ValidateAssertionInfo() error = %v, wantErr %v", tc.name, err, tc.wantErr)
			}
		})
	}
}

func Test_SAML2ACS_GetAssertionData(t *testing.T) {
	metadata, err := reader.ReadFromAttrFile("saml2", "", "testdata/idp-metadata.xml")
	if err != nil || metadata == nil {
		t.Fatal("Expected a metadata object")
	}
	sa, err := ac.NewSAML2ACS(metadata, "test", "http://www.examle.org/saml/acs", "my-sp-entity-id", []string{"memberOf"})
	if err != nil || sa == nil {
		t.Fatal("Expected a saml acs object")
	}

	values := saml2.Values{
		"displayName": types.Attribute{
			Name: "displayName",
			Values: []types.AttributeValue{
				{
					Value: "John Doe",
				},
				{
					Value: "Jane Doe",
				},
			},
		},
		"memberOf": types.Attribute{
			Name: "memberOf",
			Values: []types.AttributeValue{
				{
					Value: "group1",
				},
				{
					Value: "group2",
				},
			},
		},
	}
	var authnStatement types.AuthnStatement
	err = xml.Unmarshal([]byte(`<AuthnStatement xmlns="urn:oasis:names:tc:SAML:2.0:assertion" SessionNotOnOrAfter="2020-11-13T17:06:00Z"/>`), &authnStatement)
	if err != nil {
		t.Fatal(err)
	}

	type testCase struct {
		name          string
		assertionInfo *saml2.AssertionInfo
		want          map[string]interface{}
	}
	for _, tc := range []testCase{
		{
			"without exp",
			&saml2.AssertionInfo{
				NameID: "abc12345",
				Values: values,
			},
			map[string]interface{}{
				"sub": "abc12345",
				"attributes": map[string]interface{}{
					"displayName": "Jane Doe",
					"memberOf": []string{
						"group1",
						"group2",
					},
				},
			},
		},
		{
			"with exp",
			&saml2.AssertionInfo{
				NameID:              "abc12345",
				SessionNotOnOrAfter: authnStatement.SessionNotOnOrAfter,
				Values:              values,
			},
			map[string]interface{}{
				"sub": "abc12345",
				"exp": 1605287160,
				"attributes": map[string]interface{}{
					"displayName": "Jane Doe",
					"memberOf": []string{
						"group1",
						"group2",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertionData := sa.GetAssertionData(tc.assertionInfo)
			if fmt.Sprint(assertionData) != fmt.Sprint(tc.want) {
				t.Errorf("%s: GetAssertionData() data = %v, want %v", tc.name, assertionData, tc.want)
			}
		})
	}
}
