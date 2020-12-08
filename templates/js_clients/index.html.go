// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package js_clients

const INDEX_HTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>{{.ProjectName}}</title>
    <script src="./main.js"></script>
</head>
<body>
<h1>Welcome to your {{.ProjectName}} project!</h1>
<h3>Right click - Inspect - Console</h3>
</body>
</html>
`