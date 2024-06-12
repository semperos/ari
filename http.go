package ari

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"codeberg.org/anaseto/goal"
	"github.com/go-resty/resty/v2"
)

type HTTPClient struct {
	client *resty.Client
}

// LessT implements goal.BV.
func (httpClient *HTTPClient) LessT(y goal.BV) bool {
	// Goal falls back to ordering by type name,
	// and there is no other reasonable way to order
	// these HttpClient structs.
	return httpClient.Type() < y.Type()
}

// Matches implements goal.BV.
func (httpClient *HTTPClient) Matches(y goal.BV) bool {
	switch yv := y.(type) {
	case *HTTPClient:
		return httpClient.client == yv.client
	default:
		return false
	}
}

// Type implements goal.BV.
func (httpClient *HTTPClient) Type() string {
	return "ari.HttpClient"
}

// Append implements goal.BV.
func (httpClient *HTTPClient) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	// Go prints nil as `<nil>` so following suit.
	return append(dst, fmt.Sprintf("<%v %#v>", httpClient.Type(), httpClient.client)...)
}

func newHttpClient(optionsD *goal.D) (*HTTPClient, error) {
	// [DONE] BaseURL               string
	// [DONE] QueryParam            url.Values //  type Values map[string][]string
	// [DONE] FormData              url.Values
	// [DONE] PathParams            map[string]string
	// [DONE] RawPathParams         map[string]string
	// [DONE] Header                http.Header // Use Add methods; accept dictionary of either single strings or []string
	// [DONE] HeaderAuthorizationKey string
	// [DONE] UserInfo              *User // Struct of Username, Password string
	// [DONE] Token                 string
	// [DONE] AuthScheme            string
	// Cookies               []*http.Cookie // Medium-sized struct
	// Error                 reflect.Type
	// [DONE] Debug                 bool
	// [DONE] DisableWarn           bool
	// [DONE] AllowGetMethodPayload bool
	// [DONE] RetryCount            int
	// [DONE] RetryWaitTime         time.Duration // Pick canonical unit (millis/micros) int64
	// [DONE] RetryMaxWaitTime      time.Duration int64
	// RetryConditions       []RetryConditionFunc // Research: How tough is it to invoke a Goal lambda from Go land?
	// RetryHooks            []OnRetryFunc
	// RetryAfter            RetryAfterFunc
	// [DONE] RetryResetReaders     bool
	// JSONMarshal           func(v interface{}) ([]byte, error)
	// JSONUnmarshal         func(data []byte, v interface{}) error
	// XMLMarshal            func(v interface{}) ([]byte, error)
	// XMLUnmarshal          func(data []byte, v interface{}) error
	restyClient := resty.New()
	if optionsD.Len() == 0 {
		return &HTTPClient{resty.New()}, nil
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
					return nil, fmt.Errorf("http.client expects \"AllowGetMethodPayload\" to be 0 or 1 (falsey/truthy), but received: %v\n", value)
				}
			case "AuthScheme":
				switch goalV := value.BV().(type) {
				case goal.S:
					restyClient.AuthScheme = string(goalV)
				default:
					return nil, fmt.Errorf("http.client expects \"AuthScheme\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "BaseUrl":
				switch goalV := value.BV().(type) {
				case goal.S:
					restyClient.BaseURL = string(goalV)
				default:
					return nil, fmt.Errorf("http.client expects \"BaseUrl\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "Debug":
				switch {
				case value.IsTrue():
					restyClient.Debug = true
				case value.IsFalse():
					restyClient.Debug = false
				default:
					return nil, fmt.Errorf("http.client expects \"Debug\" to be 0 or 1, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "DisableWarn":
				switch {
				case value.IsTrue():
					restyClient.DisableWarn = true
				case value.IsFalse():
					restyClient.DisableWarn = false
				default:
					return nil, fmt.Errorf("http.client expects \"DisableWarn\" to be 0 or 1 (falsey/truthy), but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "FormData":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					formDataKeys := goalV.KeyArray()
					formDataValues := goalV.ValueArray()
					switch fdks := formDataKeys.(type) {
					case (*goal.AS):
						urlValues := make(url.Values, fdks.Len())
						for hvi := 0; hvi < formDataValues.Len(); hvi++ {
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
									return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
								}
							}
						}
						restyClient.FormData = urlValues
					default:
						return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(fdks), fdks)
					}
				default:
					return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "Header":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					headerKeys := goalV.KeyArray()
					headerValues := goalV.ValueArray()
					switch hks := headerKeys.(type) {
					case (*goal.AS):
						hd := make(http.Header, hks.Len())
						for hvi := 0; hvi < headerValues.Len(); hvi++ {
							for i, hk := range hks.Slice {
								headerValue := headerValues.At(i)
								switch hv := headerValue.BV().(type) {
								case (goal.S):
									hd.Add(hk, string(hv))
								case (*goal.AS):
									for _, w := range hv.Slice {
										hd.Add(hk, w)
									}
								default:
									return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
								}
							}
						}
						restyClient.Header = hd
					default:
						return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(hks), hks)
					}
				default:
					return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "HeaderAuthorizationKey":
				switch goalV := value.BV().(type) {
				case goal.S:
					restyClient.HeaderAuthorizationKey = string(goalV)
				default:
					return nil, fmt.Errorf("http.client expects \"HeaderAuthorizationKey\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
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
					return nil, fmt.Errorf("http.client expects \"PathParams\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "QueryParam":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					queryParamKeys := goalV.KeyArray()
					queryParamValues := goalV.ValueArray()
					switch qpks := queryParamKeys.(type) {
					case (*goal.AS):
						urlValues := make(url.Values, qpks.Len())
						for qpvi := 0; qpvi < queryParamValues.Len(); qpvi++ {
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
									return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
								}
							}
						}
						restyClient.QueryParam = urlValues
					default:
						return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(qpks), qpks)
					}
				default:
					return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
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
					return nil, fmt.Errorf("http.client expects \"RawPathParams\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "RetryCount":
				if value.IsI() {
					restyClient.RetryCount = int(value.I())
				} else {
					return nil, fmt.Errorf("http.client expects \"RetryCount\" to be an integer, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "RetryMaxWaitTimeMilli":
				if value.IsI() {
					restyClient.RetryMaxWaitTime = time.Duration(value.I()) * time.Millisecond
				} else {
					return nil, fmt.Errorf("http.client expects \"RetryMaxWaitTimeMilli\" to be an integer, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "RetryResetReaders":
				switch {
				case value.IsTrue():
					restyClient.RetryResetReaders = true
				case value.IsFalse():
					restyClient.RetryResetReaders = false
				default:
					return nil, fmt.Errorf("http.client expects \"RetryResetReaders\" to be 0 or 1 (falsey/truthy), but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "RetryWaitTimeMilli":
				switch {
				case value.IsI():
					restyClient.RetryWaitTime = time.Duration(value.I()) * time.Millisecond
				default:
					return nil, fmt.Errorf("http.client expects \"RetryWaitTimeMilli\" to be an integer, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "Token":
				switch goalV := value.BV().(type) {
				case goal.S:
					restyClient.Token = string(goalV)
				default:
					return nil, fmt.Errorf("http.client expects \"Token\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
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
							return nil, fmt.Errorf("http.client expects \"UserInfo\" to be a dictionary with string values, but received a %v: %v\n", reflect.TypeOf(uivs), uivs)
						}
					default:
						return nil, fmt.Errorf("http.client expects \"UserInfo\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(uiks), uiks)
					}
				default:
					return nil, fmt.Errorf("http.client expects \"UserInfo\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			default:
				return nil, fmt.Errorf("unsupported ari.HttpClient option: %v", k)
			}
		}
	default:
		return nil, fmt.Errorf("http.client expects a Goal dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(va), va)
	}
	return &HTTPClient{client: restyClient}, nil
}

func stringMapFromGoalDict(d *goal.D) (map[string]string, error) {
	ka := d.KeyArray()
	va := d.ValueArray()
	m := make(map[string]string, ka.Len())
	switch kas := ka.(type) {
	case *goal.AS:
		switch vas := va.(type) {
		case *goal.AS:
			vasSlice := vas.Slice
			for i, k := range kas.Slice {
				m[k] = vasSlice[i]
			}
		default:
			return nil, fmt.Errorf("[Developer Error] stringMapFromGoalDict expects a Goal dict with string keys and string values, but received values: %v\n", va)
		}
	default:
		return nil, fmt.Errorf("[Developer Error] stringMapFromGoalDict expects a Goal dict with string keys and string values, but received keys: %v\n", ka)
	}
	return m, nil
}

func VFHttpClient(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	clientOptions, ok := x.BV().(*goal.D)
	switch len(args) {
	case 1:
		if !ok {
			return panicType("http.client d", "d", x)
		}
		hc, err := newHttpClient(clientOptions)
		if err != nil {
			return goal.NewPanicError(err)
		}
		return goal.NewV(hc)
	default:
		return goal.NewPanic("http.client : too many arguments")
	}
}

func VFHTTPMaker(method string) func(goalContext *goal.Context, args []goal.V) goal.V {
	methodLower := strings.ToLower(method)
	methodUpper := strings.ToUpper(method)
	return func(_ *goal.Context, args []goal.V) goal.V {
		x := args[len(args)-1]
		switch len(args) {
		case monadic:
			url, ok := x.BV().(goal.S)
			if !ok {
				return panicType(fmt.Sprintf("http.%s s", methodLower), "s", x)
			}
			httpClient, err := newHttpClient(&goal.D{})
			if err != nil {
				return goal.NewPanicError(err)
			}
			req := httpClient.client.R()
			resp, err := req.Execute(methodUpper, string(url))
			if err != nil {
				fmt.Fprintf(os.Stderr, "HTTP error: %v\n", err)
				// Continue
				// error response built below, with
				// "ok" as false
			}
			// Construct goal.V values for return dict
			return goalDictFromResponse(resp)
		case dyadic:
			var httpClient *HTTPClient
			switch clientOpts := x.BV().(type) {
			case *HTTPClient:
				httpClient = clientOpts
			case *goal.D:
				var err error
				httpClient, err = newHttpClient(clientOpts)
				if err != nil {
					return goal.NewPanicError(err)
				}
			default:
				return goal.NewPanic(fmt.Sprintf("client http.%s url : client must be a dict or HttpClient instance, but received a %v: %v", methodLower, reflect.TypeOf(clientOpts), clientOpts))
			}
			y := args[0]
			urlS, ok := y.BV().(goal.S)
			if !ok {
				return panicType(fmt.Sprintf("HttpClient http.%s url", methodLower), "url", y)
			}
			req := httpClient.client.R()
			resp, err := req.Execute(methodUpper, string(urlS))
			if err != nil {
				fmt.Fprintf(os.Stderr, "HTTP error: %v\n", err)
				// Continue
				// error response built below, with
				// "ok" as false
			}
			return goalDictFromResponse(resp)
		case triadic:
			var httpClient *HTTPClient
			switch clientOpts := x.BV().(type) {
			case *HTTPClient:
				httpClient = clientOpts
			case *goal.D:
				var err error
				httpClient, err = newHttpClient(clientOpts)
				if err != nil {
					return goal.NewPanicError(err)
				}
			default:
				return goal.NewPanic(fmt.Sprintf("client http.%s url optionsDict : client must be a dict or HttpClient instance, but received a %v: %v", methodLower, reflect.TypeOf(clientOpts), clientOpts))
			}
			y := args[1]
			urlS, ok := y.BV().(goal.S)
			if !ok {
				return panicType(fmt.Sprintf("HttpClient http.%s url", methodLower), "url", y)
			}
			z := args[0]
			optionsD, ok := z.BV().(*goal.D)
			if !ok {
				return panicType(fmt.Sprintf("http.%s[HttpClient;url;optionsDict]", methodLower), "optionsDict", z)
			}
			req := httpClient.client.R()
			req, err := augmentRequestWithOptions(req, optionsD)
			if err != nil {
				return goal.NewPanicError(err)
			}
			resp, err := req.Execute(methodUpper, string(urlS))
			if err != nil {
				fmt.Fprintf(os.Stderr, "HTTP error: %v\n", err)
				// Continue
				// error response built below, with
				// "ok" as false
			}
			return goalDictFromResponse(resp)
		default:
			return goal.Panicf("http.%s : too many arguments (%d), expects 1, 2, or 3 arguments", methodLower, len(args))
		}
	}
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
	// TODO Consider whether this should be a pointer.
	return goal.NewD(ks, vs)
}

func augmentRequestWithOptions(req *resty.Request, optionsD *goal.D) (*resty.Request, error) {
	optionsKeys := optionsD.KeyArray()
	optionsValues := optionsD.ValueArray()
	switch kas := optionsKeys.(type) {
	case (*goal.AS):
		for i, k := range kas.Slice {
			value := optionsValues.At(i)
			switch k {
			case "QueryParam":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					queryParamKeys := goalV.KeyArray()
					queryParamValues := goalV.ValueArray()
					switch qpks := queryParamKeys.(type) {
					case (*goal.AS):
						urlValues := make(url.Values, qpks.Len())
						for qpvi := 0; qpvi < queryParamValues.Len(); qpvi++ {
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
									return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
								}
							}
						}
						req.QueryParam = urlValues
					default:
						return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(qpks), qpks)
					}
				default:
					return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "FormData":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					formDataKeys := goalV.KeyArray()
					formDataValues := goalV.ValueArray()
					switch fdks := formDataKeys.(type) {
					case (*goal.AS):
						urlValues := make(url.Values, fdks.Len())
						for hvi := 0; hvi < formDataValues.Len(); hvi++ {
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
									return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
								}
							}
						}
						req.FormData = urlValues
					default:
						return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(fdks), fdks)
					}
				default:
					return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "Header":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					headerKeys := goalV.KeyArray()
					headerValues := goalV.ValueArray()
					switch hks := headerKeys.(type) {
					case (*goal.AS):
						hd := make(http.Header, hks.Len())
						for hvi := 0; hvi < headerValues.Len(); hvi++ {
							for i, hk := range hks.Slice {
								headerValue := headerValues.At(i)
								switch hv := headerValue.BV().(type) {
								case (goal.S):
									hd.Add(hk, string(hv))
								case (*goal.AS):
									for _, w := range hv.Slice {
										hd.Add(hk, w)
									}
								default:
									return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
								}
							}
						}
						req.Header = hd
					default:
						return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(hks), hks)
					}
				default:
					return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "Cookies":
				panic("not yet implemented")
			case "PathParams":
				switch goalV := value.BV().(type) {
				case *goal.D:
					pathParams, err := stringMapFromGoalDict(goalV)
					if err != nil {
						return nil, err
					}
					req.PathParams = pathParams
				default:
					return nil, fmt.Errorf("http.client expects \"PathParams\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
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
					return nil, fmt.Errorf("http.client expects \"RawPathParams\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "Debug":
				switch {
				case value.IsTrue():
					req.Debug = true
				case value.IsFalse():
					req.Debug = false
				default:
					return nil, fmt.Errorf("http.client expects \"Debug\" to be 0 or 1, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			default:
				return nil, fmt.Errorf("unsupported resty.Request option: %v", k)
			}
		}
	default:
		return nil, fmt.Errorf("http.client expects a Goal dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(kas), kas)
	}
	return req, nil
}
