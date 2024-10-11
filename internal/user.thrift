/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

namespace go user

include "openapi.thrift"

enum Code {
    Success = 1,
    ParamInvalid = 2,
    DBErr = 3,
}

enum Gender {
    Unknown = 0,
    Male = 1,
    Female = 2,
}

struct User {
    1: required i64 user_id,
    2: required string name (openapi.property = '{
              title: "Name",
              description: "Name",
              type: "string",
              min_length: 1,
              max_length: 50
          }'),
    3: required Gender gender,
    4: required i64 age,
    5: required string introduce,
}

struct CreateUserRequest {
    1: required string name,
    2: required Gender gender,
    3: required i64 age,
    4: required string introduce,
}

struct CreateUserResponse {
    1: required Code code,
    2: required string msg,
}

struct QueryUserRequest {
    1: optional string Keyword,
    2: required i64 page,
    3: required i64 page_size,
}

struct QueryUserResponse {
    1: required Code code,
    2: required string msg,
    3: required list<User> users,
    4: required i64 total,
}

struct DeleteUserRequest {
    //用户编号
    1: required i64 user_id,
}

struct DeleteUserResponse {
    1: required Code code,
    2: required string msg,
}

struct UpdateUserRequest {
    1: required i64 user_id,
    2: required string name,
    //性别
    3: required Gender gender,
    4: required i64 age,
    5: required string introduce,
}

struct UpdateUserResponse {
    1: required Code code,
    2: required string msg (openapi.property = '{
            title: "response content",
            description: "response content",
            type: "string",
            min_length: 1,
            max_length: 80
        }'),
} (openapi.schema = '{
       title: "Hello - response",
       description: "Hello - response",
       required: [
          "msg"
       ]
    }')

// UserService描述
service UserService {
    UpdateUserResponse UpdateUser(1: UpdateUserRequest req),
    DeleteUserResponse DeleteUser(1: DeleteUserRequest req),
    QueryUserResponse QueryUser(1: QueryUserRequest req),
    CreateUserResponse CreateUser(1: CreateUserRequest req),
} (api.base_domain = "127.0.0.1:8888", openapi.document = '{
        info: {
           title: "kitex example swagger doc",
           version: "0.0.1"
        }
     }')