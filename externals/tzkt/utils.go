package tzkt

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/bitmark-inc/nft-indexer/traceutils"
)

func IgnoreBigTotalSupply(resString string, totalSupply string) (string, error) {
	var result string

	subString := `"totalSupply":"` + totalSupply + `"`
	i := strings.Index(resString, subString)

	if i < 0 {
		return "", errors.New("can not find bigInt total supply")
	}

	replaceString := `"totalSupply":"` + "-1" + `"`
	result = resString[:i] + replaceString + resString[i+len(subString):]

	return result, nil
}

func UnmarshalJSON(req *http.Request, resp *http.Response, responseData interface{}) error {
	errResp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	resString := string(errResp)

	for true {
		err := json.Unmarshal([]byte(resString), &responseData)

		if err != nil {
			if strings.Contains(err.Error(), "json: cannot unmarshal number") && strings.Contains(err.Error(), "Token.token.totalSupply") {
				resString, _ = IgnoreBigTotalSupply(resString, strings.Split(err.Error(), " ")[4])

				continue
			}

			logrus.
				WithField("req", traceutils.DumpRequest(req)).
				WithField("resp", traceutils.DumpResponse(resp)).
				Error("tzkt error response")

			return err
		}
		break
	}
	return nil
}
