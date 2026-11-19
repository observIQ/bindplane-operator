/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

// Shared literal values used across controller tests.
const (
	testNamespaceDefault      = "default"
	testValue                 = "test"
	testValueTrueTitle        = "True"
	testValueFalse            = "false"
	testUserLabel             = "user-label"
	testUserValue             = "user-value"
	testBindplaneName         = "test-bp"
	testMyBindplaneName       = "my-bp"
	testLicense               = "license"
	testTLSSecretName         = "tls-secret"
	testPostgresTLSSecretName = "pg-tls"
	testIssuerName            = "my-issuer"
	testNatsIssuerName        = "nats-issuer"
	testTAIssuerName          = "ta-issuer"
	testCAIssuerName          = "ca-issuer"
	testVMHost                = "vm.example.internal"
	testGroupsClaim           = "my_groups"
	testOpVar                 = "OP_VAR"
	testHTTPProxyEnv          = "HTTP_PROXY"
	testNoProxyEnv            = "NO_PROXY"
	testRedisCAName           = "redis-ca"
	testUserVolumeName        = "user-vol"
	testOpMountPath           = "/op"
)
