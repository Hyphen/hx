package members

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type MemberServicer interface {
	ListMembers(orgID string) ([]Member, error)
	CreateMemberForOrg(orgID string, member Member) (Member, error)
	DeleteMember(orgID, memberID string) error
}

type MemberService struct {
	baseUrl    string
	httpClient httputil.Client
}

func NewService() *MemberService {
	baseUrl := apiconf.GetBaseApixUrl()
	return &MemberService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}

func (ms *MemberService) ListMembers(orgID string) ([]Member, error) {

	url := fmt.Sprintf("%s/api/organizations/%s/members/", ms.baseUrl, orgID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create request")
	}

	resp, err := ms.httpClient.Do(req)
	if err != nil {
		return nil, err
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

	url := fmt.Sprintf("%s/api/organizations/%s/members/", ms.baseUrl, orgID)

	payload, err := json.Marshal(member)
	if err != nil {
		return Member{}, errors.Wrap(err, "Failed to marshal member data")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return Member{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := ms.httpClient.Do(req)
	if err != nil {
		return Member{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
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

	url := fmt.Sprintf("%s/api/organizations/%s/members/%s/", ms.baseUrl, orgID, memberID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create request")
	}

	resp, err := ms.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp == nil {
		return errors.New("Received nil response")
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.HandleHTTPError(resp)
	}

	return nil
}
