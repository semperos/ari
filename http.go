package ari

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"codeberg.org/anaseto/goal"
	"github.com/go-resty/resty/v2"
)

type HTTPClient struct {
	Client *resty.Client
}

// LessT implements goal.BV.
func (httpClient *HTTPClient) LessT(y goal.BV) bool {
	// Goal falls back to ordering by type name,
	// and there is no other reasonable way to order
	// these HTTPClient structs.
	return httpClient.Type() < y.Type()
}

// Matches implements goal.BV.
func (httpClient *HTTPClient) Matches(y goal.BV) bool {
	switch yv := y.(type) {
	case *HTTPClient:
		return httpClient.Client == yv.Client
	default:
		return false
	}
}

// Type implements goal.BV.
func (httpClient *HTTPClient) Type() string {
	return "ari.HTTPClient"
}

// Append implements goal.BV.
func (httpClient *HTTPClient) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	// Go prints nil as `<nil>` so following suit.
	return append(dst, fmt.Sprintf("<%v %#v>", httpClient.Type(), httpClient.Client)...)
}

//nolint:cyclop,funlen,gocognit,gocyclo // No code shared, ball of wax stays together.
func NewHTTPClient(optionsD *goal.D) (*HTTPClient, error) {
	// Not currently implemented:
	// Cookies               []*http.Cookie // Medium-sized struct
	// Error                 reflect.Type
	// RetryConditions       []RetryConditionFunc // Research: How tough is it to invoke a Goal lambda from Go land?
	// RetryHooks            []OnRetryFunc
	// RetryAfter            RetryAfterFunc
	// JSONMarshal           func(v interface{}) ([]byte, error)
	// JSONUnmarshal         func(data []byte, v interface{}) error
	// XMLMarshal            func(v interface{}) ([]byte, error)
	// XMLUnmarshal          func(data []byte, v interface{}) error
	goalFnName := "http.client"
	restyClient := resty.New()
	httpClient := HTTPClient{restyClient}
	if optionsD.Len() == 0 {
		return &httpClient, nil
	}
	ka := optionsD.KeyArray()
	va := optionsD.ValueArray()
	switch kas := ka.(type) {
	case *goal.AS:
		for i, k := range kas.Slice {
			value := va.At(i)
			switch k {
			case "AllowGetMethodPayload":
				switch {
				case value.IsTrue():
					restyClient.AllowGetMethodPayload = true
				case value.IsFalse():
					restyClient.AllowGetMethodPayload = false
				default:
					return nil, fmt.Errorf("%v expects \"AllowGetMethodPayload\" "+
						"to be 0 or 1 (falsey/truthy), but received: %v",
						goalFnName,
						value)
				}
			case "AuthScheme":
				switch goalV := value.BV().(type) {
				case goal.S:
					restyClient.AuthScheme = string(goalV)
				default:
					return nil, fmt.Errorf("%v expects \"AuthScheme\" "+
						"to be a string, but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "BaseURL":
				switch goalV := value.BV().(type) {
				case goal.S:
					restyClient.BaseURL = string(goalV)
				default:
					return nil, fmt.Errorf("%v expects \"BaseURL\" "+
						"to be a string, but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "Debug":
				switch {
				case value.IsTrue():
					restyClient.Debug = true
				case value.IsFalse():
					restyClient.Debug = false
				default:
					return nil, fmt.Errorf("%v expects \"Debug\" to be 0 or 1, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "DisableWarn":
				switch {
				case value.IsTrue():
					restyClient.DisableWarn = true
				case value.IsFalse():
					restyClient.DisableWarn = false
				default:
					return nil, fmt.Errorf("%v expects \"DisableWarn\" "+
						"to be 0 or 1 (falsey/truthy), but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "FormData":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					urlValues, err := processFormData(goalV, goalFnName)
					if err != nil {
						return nil, err
					}
					restyClient.FormData = urlValues
				default:
					return nil, fmt.Errorf("%v expects \"FormData\" to be a dictionary, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "Header":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					header, err := processHeader(goalV, goalFnName)
					if err != nil {
						return nil, err
					}
					restyClient.Header = header
				default:
					return nil, fmt.Errorf("%v expects \"Header\" to be a dictionary, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "HeaderAuthorizationKey":
				switch goalV := value.BV().(type) {
				case goal.S:
					restyClient.HeaderAuthorizationKey = string(goalV)
				default:
					return nil, fmt.Errorf("%v expects \"HeaderAuthorizationKey\" to be a string, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "PathParams":
				switch goalV := value.BV().(type) {
				case *goal.D:
					pathParams, err := stringMapFromGoalDict(goalV)
					if err != nil {
						return nil, err
					}
					restyClient.PathParams = pathParams
				default:
					return nil, fmt.Errorf("%v expects \"PathParams\" to be a string, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "QueryParam":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					urlValues, err := processQueryParam(goalV, goalFnName)
					if err != nil {
						return nil, err
					}
					restyClient.QueryParam = urlValues
				default:
					return nil, fmt.Errorf("%v expects \"QueryParam\" to be a dictionary, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "RawPathParams":
				switch goalV := value.BV().(type) {
				case *goal.D:
					pathParams, err := stringMapFromGoalDict(goalV)
					if err != nil {
						return nil, err
					}
					restyClient.RawPathParams = pathParams
				default:
					return nil, fmt.Errorf("%v expects \"RawPathParams\" to be a string, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "RetryCount":
				if value.IsI() {
					restyClient.RetryCount = int(value.I())
				} else {
					return nil, fmt.Errorf("%v expects \"RetryCount\" to be an integer, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "RetryMaxWaitTimeMilli":
				if value.IsI() {
					restyClient.RetryMaxWaitTime = time.Duration(value.I()) * time.Millisecond
				} else {
					return nil, fmt.Errorf("%v expects \"RetryMaxWaitTimeMilli\" to be an integer, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "RetryResetReaders":
				switch {
				case value.IsTrue():
					restyClient.RetryResetReaders = true
				case value.IsFalse():
					restyClient.RetryResetReaders = false
				default:
					return nil, fmt.Errorf("%v expects \"RetryResetReaders\" to be 0 or 1 (falsey/truthy), "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "RetryWaitTimeMilli":
				switch {
				case value.IsI():
					restyClient.RetryWaitTime = time.Duration(value.I()) * time.Millisecond
				default:
					return nil, fmt.Errorf("%v expects \"RetryWaitTimeMilli\" to be an integer, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "Token":
				switch goalV := value.BV().(type) {
				case goal.S:
					restyClient.Token = string(goalV)
				default:
					return nil, fmt.Errorf("%v expects \"Token\" to be a string, but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "UserInfo":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					userInfoKeys := goalV.KeyArray()
					userInfoValues := goalV.ValueArray()
					switch uiks := userInfoKeys.(type) {
					case (*goal.AS):
						switch uivs := userInfoValues.(type) {
						case (*goal.AS):
							userInfo := resty.User{}
							for i, uik := range uiks.Slice {
								switch uik {
								case "Username":
									userInfo.Username = uivs.Slice[i]
								case "Password":
									userInfo.Password = uivs.Slice[i]
								default:
									return nil, fmt.Errorf("unsupported \"UserInfo\" key: %v", uik)
								}
							}
							restyClient.UserInfo = &userInfo
						default:
							return nil, fmt.Errorf("%v expects \"UserInfo\" to be a dictionary "+
								"with string values, but received a %v: %v",
								goalFnName,
								reflect.TypeOf(uivs),
								uivs)
						}
					default:
						return nil, fmt.Errorf("%v expects \"UserInfo\" to be a dictionary "+
							"with string keys, but received a %v: %v",
							goalFnName,
							reflect.TypeOf(uiks),
							uiks)
					}
				default:
					return nil, fmt.Errorf("%v expects \"UserInfo\" to be a dictionary, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			default:
				return nil, fmt.Errorf("unsupported ari.HTTPClient option: %v", k)
			}
		}
	default:
		return nil, fmt.Errorf("%v expects a Goal dictionary with string keys, "+
			"but received a %v: %v",
			goalFnName,
			reflect.TypeOf(va),
			va)
	}
	return &httpClient, nil
}

func VFHTTPClientFn() func(goalContext *goal.Context, args []goal.V) goal.V {
	return func(_ *goal.Context, args []goal.V) goal.V {
		x := args[len(args)-1]
		clientOptions, ok := x.BV().(*goal.D)
		switch len(args) {
		case 1:
			if !ok {
				return panicType("http.client d", "d", x)
			}
			hc, err := NewHTTPClient(clientOptions)
			if err != nil {
				return goal.NewPanicError(err)
			}
			return goal.NewV(hc)
		default:
			return goal.NewPanic("http.client : too many arguments")
		}
	}
}

func VFHTTPMaker(ariContext *Context, method string) func(goalContext *goal.Context, args []goal.V) goal.V {
	methodLower := strings.ToLower(method) // Used for function name
	methodUpper := strings.ToUpper(method) // Used by go-resty for HTTP method
	return func(_ *goal.Context, args []goal.V) goal.V {
		x := args[len(args)-1]
		switch len(args) {
		case monadic:
			return httpMakerMonadic(ariContext, x, methodLower, methodUpper)
		case dyadic:
			return httpMakerDyadic(x, args, methodLower, methodUpper)
		case triadic:
			return httpMakerTriadic(x, args, methodLower, methodUpper)
		default:
			return goal.Panicf("http.%s : too many arguments (%d), expects 1, 2, or 3 arguments", methodLower, len(args))
		}
	}
}

func httpMakerMonadic(ariContext *Context, x goal.V, methodLower string, methodUpper string) goal.V {
	url, ok := x.BV().(goal.S)
	if !ok {
		return panicType(fmt.Sprintf("http.%s s", methodLower), "s", x)
	}
	// Cache the client for the monadic arity.
	if ariContext.HTTPClient == nil {
		// Saves in ariContext.HTTPClient
		httpClient, err := NewHTTPClient(goalNewDictEmpty())
		if err != nil {
			return goal.NewPanicError(err)
		}
		// NB: Keep pointer to last-minted HTTPClient, so tests can mock and override.
		ariContext.HTTPClient = httpClient
	}
	req := ariContext.HTTPClient.Client.R()
	resp, err := req.Execute(methodUpper, string(url))
	if err != nil {
		return goal.Errorf("HTTP ERROR: %v", err)
	}
	return goalDictFromResponse(resp)
}

func httpMakerDyadic(x goal.V, args []goal.V, methodLower string, methodUpper string) goal.V {
	var httpClient *HTTPClient
	switch clientOpts := x.BV().(type) {
	case *HTTPClient:
		httpClient = clientOpts
	case *goal.D:
		var err error
		httpClient, err = NewHTTPClient(clientOpts)
		if err != nil {
			return goal.NewPanicError(err)
		}
	default:
		errMsg := fmt.Sprintf("client http.%s url : client must be a dict or ari.HTTPClient instance, "+
			"but received a %v: %v",
			methodLower,
			reflect.TypeOf(clientOpts),
			clientOpts)
		return goal.NewPanic(errMsg)
	}
	y := args[0]
	urlS, ok := y.BV().(goal.S)
	if !ok {
		return panicType(fmt.Sprintf("ari.HTTPClient http.%s url", methodLower), "url", y)
	}
	req := httpClient.Client.R()
	resp, err := req.Execute(methodUpper, string(urlS))
	if err != nil {
		return goal.Errorf("HTTP ERROR: %v", err)
	}
	return goalDictFromResponse(resp)
}

func httpMakerTriadic(x goal.V, args []goal.V, methodLower string, methodUpper string) goal.V {
	var httpClient *HTTPClient
	switch clientOpts := x.BV().(type) {
	case *HTTPClient:
		httpClient = clientOpts
	case *goal.D:
		var err error
		httpClient, err = NewHTTPClient(clientOpts)
		if err != nil {
			return goal.NewPanicError(err)
		}
	default:
		errMsg := fmt.Sprintf("client http.%s url optionsDict : client must be a dict or ari.HTTPClient instance, "+
			"but received a %v: %v",
			methodLower,
			reflect.TypeOf(clientOpts),
			clientOpts)
		return goal.NewPanic(errMsg)
	}
	y := args[1]
	urlS, ok := y.BV().(goal.S)
	if !ok {
		return panicType(fmt.Sprintf("ari.HTTPClient http.%s url", methodLower), "url", y)
	}
	z := args[0]
	optionsD, ok := z.BV().(*goal.D)
	if !ok {
		return panicType(fmt.Sprintf("http.%s[ari.HTTPClient;url;optionsDict]", methodLower), "optionsDict", z)
	}
	req := httpClient.Client.R()
	req, err := augmentRequestWithOptions(req, optionsD, methodLower)
	if err != nil {
		return goal.NewPanicError(err)
	}
	resp, err := req.Execute(methodUpper, string(urlS))
	if err != nil {
		return goal.Errorf("HTTP ERROR: %v", err)
	}
	return goalDictFromResponse(resp)
}

func goalDictFromResponse(resp *resty.Response) goal.V {
	statusS := goal.NewS(resp.Status())
	headers := resp.Header()
	headerKeysSlice := make([]string, 0)
	headerValuesSlice := make([]goal.V, 0)
	for k, vs := range headers {
		headerKeysSlice = append(headerKeysSlice, k)
		valuesAS := goal.NewAS(vs)
		headerValuesSlice = append(headerValuesSlice, valuesAS)
	}
	headerD := goal.NewD(goal.NewAS(headerKeysSlice), goal.NewAV(headerValuesSlice))
	bodyS := goal.NewS(resp.String())
	var isOk goal.V
	if resp.IsSuccess() {
		isOk = goal.NewI(1)
	} else {
		isOk = goal.NewI(0)
	}
	ks := goal.NewAS([]string{"status", "headers", "string", "ok"})
	vs := goal.NewAV([]goal.V{statusS, headerD, bodyS, isOk})
	return goal.NewD(ks, vs)
}

//nolint:cyclop,funlen,gocognit // every case edits the same req, returned at end
func augmentRequestWithOptions(req *resty.Request, optionsD *goal.D, methodLower string) (*resty.Request, error) {
	goalFnName := "http." + methodLower
	optionsKeys := optionsD.KeyArray()
	optionsValues := optionsD.ValueArray()
	switch kas := optionsKeys.(type) {
	case (*goal.AS):
		for i, k := range kas.Slice {
			value := optionsValues.At(i)
			switch k {
			case "Body":
				switch goalV := value.BV().(type) {
				case *goal.AB:
					req.SetBody(goalV.Slice)
				case goal.S:
					req.SetBody(string(goalV))
				}
			case "Cookies":
				panic("not yet implemented")
			case "Debug":
				switch {
				case value.IsTrue():
					req.Debug = true
				case value.IsFalse():
					req.Debug = false
				default:
					return nil, fmt.Errorf("%v expects \"Debug\" to be 0 or 1, but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "FormData":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					urlValues, err := processFormData(goalV, goalFnName)
					if err != nil {
						return nil, err
					}
					req.FormData = urlValues
				default:
					return nil, fmt.Errorf("%v expects \"FormData\" to be a dictionary, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "Header":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					header, err := processHeader(goalV, goalFnName)
					if err != nil {
						return nil, err
					}
					req.Header = header
				default:
					return nil, fmt.Errorf("%v expects \"Header\" to be a dictionary, but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "PathParams":
				switch goalV := value.BV().(type) {
				case *goal.D:
					pathParams, err := stringMapFromGoalDict(goalV)
					if err != nil {
						return nil, err
					}
					req.PathParams = pathParams
				default:
					return nil, fmt.Errorf("%v expects \"PathParams\" to be a string, but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "QueryParam":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					urlValues, err := processQueryParam(goalV, goalFnName)
					if err != nil {
						return nil, err
					}
					req.QueryParam = urlValues
				default:
					return nil, fmt.Errorf("%v expects \"QueryParam\" to be a dictionary, "+
						"but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			case "RawPathParams":
				switch goalV := value.BV().(type) {
				case *goal.D:
					pathParams, err := stringMapFromGoalDict(goalV)
					if err != nil {
						return nil, err
					}
					req.RawPathParams = pathParams
				default:
					return nil, fmt.Errorf("%v expects \"RawPathParams\" to be a string, but received a %v: %v",
						goalFnName,
						reflect.TypeOf(value),
						value)
				}
			default:
				return nil, fmt.Errorf("unsupported resty.Request option: %v", k)
			}
		}
	default:
		return nil, fmt.Errorf("%v expects a Goal dictionary with string keys, but received a %v: %v",
			goalFnName,
			reflect.TypeOf(kas),
			kas)
	}
	return req, nil
}

//nolint:dupl // Add methods of url.Values and http.Header differ, skipping type gymnastics.
func processFormData(goalD *goal.D, goalFnName string) (url.Values, error) {
	urlValues := make(url.Values, goalD.Len())
	formDataKeys := goalD.KeyArray()
	formDataValues := goalD.ValueArray()
	switch fdks := formDataKeys.(type) {
	case (*goal.AS):
		for i, hk := range fdks.Slice {
			formDataValue := formDataValues.At(i)
			switch hv := formDataValue.BV().(type) {
			case (goal.S):
				urlValues.Add(hk, string(hv))
			case (*goal.AS):
				for _, w := range hv.Slice {
					urlValues.Add(hk, w)
				}
			default:
				return nil, fmt.Errorf("%v expects \"FormData\" "+
					"to be a dictionary with values that are strings or lists of strings, "+
					"but received a %v: %v",
					goalFnName,
					reflect.TypeOf(hv),
					hv)
			}
		}
	default:
		return nil, fmt.Errorf("%v expects \"FormData\" to be a dictionary "+
			"with string keys, but received a %v: %v",
			goalFnName,
			reflect.TypeOf(fdks),
			fdks)
	}
	return urlValues, nil
}

//nolint:dupl // Add methods of url.Values and http.Header differ, skipping type gymnastics.
func processHeader(goalD *goal.D, goalFnName string) (http.Header, error) {
	header := make(http.Header, goalD.Len())
	headerKeys := goalD.KeyArray()
	headerValues := goalD.ValueArray()
	switch hks := headerKeys.(type) {
	case (*goal.AS):
		for i, hk := range hks.Slice {
			headerValue := headerValues.At(i)
			switch hv := headerValue.BV().(type) {
			case (goal.S):
				header.Add(hk, string(hv))
			case (*goal.AS):
				for _, w := range hv.Slice {
					header.Add(hk, w)
				}
			default:
				return nil, fmt.Errorf("%v expects \"Header\" to be "+
					"a dictionary with values that are strings or lists of strings, "+
					"but received a %v: %v",
					goalFnName,
					reflect.TypeOf(hv),
					hv)
			}
		}
	default:
		return nil, fmt.Errorf("%v expects \"Header\" to be a dictionary "+
			"with string keys, but received a %v: %v",
			goalFnName,
			reflect.TypeOf(hks),
			hks)
	}
	return header, nil
}

func processQueryParam(goalD *goal.D, goalFnName string) (url.Values, error) {
	urlValues := make(url.Values, goalD.Len())
	queryParamKeys := goalD.KeyArray()
	queryParamValues := goalD.ValueArray()
	switch qpks := queryParamKeys.(type) {
	case (*goal.AS):
		for i, hk := range qpks.Slice {
			queryParamValue := queryParamValues.At(i)
			switch hv := queryParamValue.BV().(type) {
			case (goal.S):
				urlValues.Add(hk, string(hv))
			case (*goal.AS):
				for _, w := range hv.Slice {
					urlValues.Add(hk, w)
				}
			default:
				return nil, fmt.Errorf("%v expects \"QueryParam\" to be a dictionary "+
					"with values that are strings or lists of strings, but received a %v: %v",
					goalFnName,
					reflect.TypeOf(hv),
					hv)
			}
		}
	default:
		return nil, fmt.Errorf("%v expects \"QueryParam\" to be a dictionary "+
			"with string keys, but received a %v: %v",
			goalFnName,
			reflect.TypeOf(qpks),
			qpks)
	}
	return urlValues, nil
}
