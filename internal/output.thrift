namespace go "github.com/hertz-contrib/swagger-generate/example"

enum Age {
  Age1 = 0;
  b = 1;
}

struct BodyReqBody {
    1: string Body1
    2: string Body
}

struct FormReqForm {
    1: FormReq_InnerForm FormReqInnerForm
    2: string Form1
}

struct FormReq_InnerForm {
    1: string Form3
}

struct HelloRespBody {
    1: string Body
}

struct Items1Item {
    1: Age Age
    2: string Name
}

struct HelloService2QueryMethod2Request {
    1: map<string, string> additionalProperties (
        api.query = "query1"
    )
    2: list<Items1Item> Items1 (
        api.query = "items1"
    )
    3: string Query2 (
        api.query = "query2"
    )
}

struct HelloService2QueryMethod2Response200 {
    1: string Token (
        api.header = "token"
    )
    2: HelloRespBody HelloRespBody (
        api.body = "HelloRespBody"
    )
}

struct HelloService1PathMethodRequest {
    1: string Path1 (
        api.path = "path1"
    )
}

struct HelloService1PathMethodResponse200 {
    1: string Token (
        api.header = "token"
    )
    2: HelloRespBody HelloRespBody (
        api.body = "HelloRespBody"
    )
}

struct HelloService1BodyMethodRequest {
    1: BodyReqBody BodyReqBody (
        api.body = "BodyReqBody"
    )
    2: string Query2 (
        api.query = "query2"
    )
}

struct HelloService1BodyMethodResponse200 {
    1: string Token (
        api.header = "token"
    )
    2: HelloRespBody HelloRespBody (
        api.body = "HelloRespBody"
    )
}

struct HelloService1FormMethodRequest {
    1: FormReqForm FormReqForm (
        api.form = "FormReqForm"
    )
}

struct HelloService1FormMethodResponse200 {
    1: string Token (
        api.header = "token"
    )
    2: HelloRespBody HelloRespBody (
        api.body = "HelloRespBody"
    )
}

struct HelloService1QueryMethod1Request {
    1: map<string, string> additionalProperties (
        api.query = "query1"
    )
    2: list<string> Items (
        api.query = "items"
    )
    3: string Query2 (
        api.query = "query2"
    )
}

struct HelloService1QueryMethod1Response200 {
    1: string Token (
        api.header = "token"
    )
    2: HelloRespBody HelloRespBody (
        api.body = "HelloRespBody"
    )
}

service HelloService2 {
    HelloService2QueryMethod2Response200 HelloService2QueryMethod2 (1: HelloService2QueryMethod2Request req) (
        api.get = "/hello2"
    )
}

service HelloService1 {
    HelloService1PathMethodResponse200 HelloService1PathMethod (1: HelloService1PathMethodRequest req) (
        api.get = "/path:path1"
    )
    HelloService1BodyMethodResponse200 HelloService1BodyMethod (1: HelloService1BodyMethodRequest req) (
        api.post = "/body"
    )
    HelloService1FormMethodResponse200 HelloService1FormMethod (1: HelloService1FormMethodRequest req) (
        api.post = "/form"
    )
    HelloService1QueryMethod1Response200 HelloService1QueryMethod1 (1: HelloService1QueryMethod1Request req) (
        api.get = "/hello1"
    )
}

