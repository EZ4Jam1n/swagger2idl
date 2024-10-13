namespace go example

include "openapi.thrift"

enum HelloRespBodyEnum {
  BODY1 = 0;
}

struct HelloService1QueryMethod1Request {
    1: list<string> Items (api.query = "items",
    openapi.parameter = '{
       name: "items";
       in: "query";
     }')
      // QueryValue描述
    2: string Query2 (api.query = "query2",
    openapi.parameter = '{
       name: "query2";
       in: "query";
       description: "QueryValue描述";
       required: true;
     }')
}

struct HelloService1QueryMethod1Response {
    1: string Token (api.header = "token",
    openapi.property = '{
       parameter: {
       };
     }')
    2: HelloRespBody HelloRespBody (api.body = "HelloRespBody")
}

struct HelloService1PathMethodResponse {
    1: string Token (api.header = "token",
    openapi.property = '{
       parameter: {
       };
     }')
    2: HelloRespBody HelloRespBody (api.body = "HelloRespBody")
}

struct HelloService1BodyMethodRequest {
    1: string Body (openapi.property = '{
      value: {
        type: [
          "string"
        ];
        title: "this is an override field schema title";
        max_length: "255";
      };
    }',
    api.body = "Body")
      // field: query描述
    2: string Query2 (api.query = "query2",
    openapi.parameter = '{
       name: "query2";
       in: "query";
       description: "field: query描述";
     }')
}

struct HelloService1BodyMethodResponse {
    1: TokenEnum token_field (api.header = "token",
    openapi.property = '{
       parameter: {
       };
     }')
    2: HelloRespBody HelloRespBody (api.body = "HelloRespBody")
}

struct HelloService1FormMethodRequest {
    1: string Form1 (openapi.property = '{
      value: {
        type: [
          "string"
        ];
        title: "this is an override field schema title";
        max_length: "255";
      };
    }',
    api.form = "Form1")
}

struct HelloService1FormMethodResponse {
    1: string Token (api.header = "token",
    openapi.property = '{
       parameter: {
       };
     }')
    2: HelloRespBody HelloRespBody (api.body = "HelloRespBody")
}

service HelloService1 {
    HelloService1QueryMethod1Response HelloService1QueryMethod1 (1: HelloService1QueryMethod1Request req) (
        api.get = "/hello",
        openapi.operation = '{
       tags: [
         "HelloService1"
       ];
       operation_id: "HelloService1_QueryMethod1";
     }'
    )
    HelloService1PathMethodResponse HelloService1PathMethod (1:  req) (
        api.get = "/path:path1",
        openapi.operation = '{
       tags: [
         "HelloService1"
       ];
       operation_id: "HelloService1_PathMethod";
     }'
    )
    HelloService1BodyMethodResponse HelloService1BodyMethod (1: HelloService1BodyMethodRequest req) (
        api.post = "/body",
        openapi.operation = '{
       tags: [
         "HelloService1"
       ];
       operation_id: "HelloService1_BodyMethod";
     }'
    )
    HelloService1FormMethodResponse HelloService1FormMethod (1: HelloService1FormMethodRequest req) (
        api.post = "/form",
        openapi.operation = '{
       tags: [
         "HelloService1"
       ];
       operation_id: "HelloService1_FormMethod";
     }'
    )
}(openapi.document = '{
  openapi: "3.0.3";
  info: {
    title: "example swagger doc";
    description: "HelloService1描述";
    version: "Version from annotation";
  };
  servers: [
    {
      url: "http://localhost:8888";
    }
  ];
  tags: [
    {
      name: "HelloService1";
    }
  ];
}')

