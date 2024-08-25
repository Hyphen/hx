package members

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type MemberServicer interface {
	ListMembers(orgID string) ([]Member, error)
	CreateMemberForOrg(orgID string, member Member) (Member, error)
	DeleteMember(orgID, memberID string) error
}

type MemberService struct {
	baseUrl      string
	oauthService oauth.OAuthServicer
	httpClient   httputil.Client
}

func NewService() *MemberService {
	baseUrl := "https://dev-api.hyphen.ai"
	if customAPI := os.Getenv("HYPHEN_CUSTOM_APIX"); customAPI != "" {
		baseUrl = customAPI
	}
	return &MemberService{
		baseUrl:      baseUrl,
		oauthService: oauth.DefaultOAuthService(),
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

func (ms *MemberService) ListMembers(orgID string) ([]Member, error) {
	token, err := ms.oauthService.GetValidToken()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	url := fmt.Sprintf("%s/api/organizations/%s/members/", ms.baseUrl, orgID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := ms.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}

	var response struct {
		Data []Member `json:"data"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return response.Data, nil
}

func (ms *MemberService) CreateMemberForOrg(orgID string, member Member) (Member, error) {
	token, err := ms.oauthService.GetValidToken()
	if err != nil {
		return Member{}, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	url := fmt.Sprintf("%s/api/organizations/%s/members/", ms.baseUrl, orgID)

	payload, err := json.Marshal(member)
	if err != nil {
		return Member{}, errors.Wrap(err, "Failed to marshal member data")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return Member{}, errors.Wrap(err, "Failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := ms.httpClient.Do(req)
	if err != nil {
		return Member{}, errors.Wrap(err, "Failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return Member{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Member{}, errors.Wrap(err, "Failed to read response body")
	}

	var createdMember Member
	err = json.Unmarshal(body, &createdMember)
	if err != nil {
		return Member{}, errors.Wrap(err, "Failed to parse JSON response")
	}

	return createdMember, nil
}

func (ms *MemberService) DeleteMember(orgID, memberID string) error {
	token, err := ms.oauthService.GetValidToken()
	if err != nil {
		return errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	url := fmt.Sprintf("%s/api/organizations/%s/members/%s/", ms.baseUrl, orgID, memberID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := ms.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failed to send request")
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp == nil {
		return errors.New("Received nil response")
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return errors.HandleHTTPError(resp)
	}

	return nil
}
