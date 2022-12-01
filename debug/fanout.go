package debug

import (
	"fmt"
	"net/http"
	"text/template"

	pb "github.com/dfanout/dfanout/proto"
)

type Handler struct {
	fanout    string
	endpoints []*pb.Endpoint
}

func NewHandler(fanout string, e []*pb.Endpoint) *Handler {
	return &Handler{
		fanout:    fanout,
		endpoints: e,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var endpoints []endpointData
	for _, e := range h.endpoints {
		httpEndpoint := e.Destination.(*pb.Endpoint_HttpEndpoint).HttpEndpoint
		endpoints = append(endpoints, endpointData{
			Name:    e.Name,
			Primary: e.Primary,
			URL:     httpEndpoint.Url,
			Method:  httpEndpoint.Method,
			Timeout: httpEndpoint.TimeoutMs,
		})
	}
	if err := debugTmpl.Execute(w, &debugData{
		Fanout:    h.fanout,
		Endpoints: endpoints,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to render the page: %v", err)
	}
}

type endpointData struct {
	Name    string
	Primary bool
	URL     string
	Method  string
	Timeout int64
}

type debugData struct {
	Fanout    string
	Endpoints []endpointData
}

var debugTmpl = template.Must(template.New("debug").Parse(debugHTML))

const debugHTML = `
<!DOCTYPE html>
<html>
<head>
<title>dfanout: {{.Fanout}}</title>
    <link href="https://fonts.googleapis.com/css?family=Roboto:400,500,700&display=swap" rel="stylesheet">
    <meta name="viewport" content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
<!-- 
MIT License

Copyright (c) 2019 Alyssa X

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
-->
<style>
body, html {
    margin: 0px;
    padding: 30px 20px;
    background-image: url(https://i.imgur.com/hgRvtXd.png);
    background-repeat: repeat;
    background-size: 30px 30px;
    background-color: #FBFBFB;
}
.blockin {
    display: inline-block;
    vertical-align: top;
    margin-left: 12px;
}
.blockico {
    width: 36px;
    height: 36px;
    background-color: #F1F4FC;
    border-radius: 5px;
    text-align: center;
    white-space: nowrap;
}
.blockico span {
    height: 100%;
    width: 0px;
    display: inline-block;
    vertical-align: middle;
}
.blockico img {
    vertical-align: middle;
    margin-left: auto;
    margin-right: auto;
    display: inline-block;
}
.blocktext {
    display: inline-block;
    vertical-align: top;
    margin-left: 12px
}
.blocktitle {
    margin: 0px!important;
    padding: 0px!important;
    font-family: Roboto;
    font-weight: 500;
    font-size: 16px;
    color: #393C44;
}
.blockdesc {
    margin-top: 5px;
    font-family: Roboto;
    color: #808292;
    font-size: 14px;
    line-height: 21px;
}
.blockyname {
    font-family: Roboto;
    font-weight: 500;
    color: #253134;
    display: inline-block;
    vertical-align: middle;
    margin-left: 8px;
    font-size: 16px;
}
.blockyleft img {
    display: inline-block;
    vertical-align: middle;
}
.blockyright {
    display: inline-block;
    float: right;
    vertical-align: middle;
    margin-right: 20px;
    margin-top: 10px;
    width: 28px;
    height: 28px;
    border-radius: 5px;
    text-align: center; 
    background-color: #FFF;
    transition: all .3s cubic-bezier(.05,.03,.35,1);
    z-index: 10;
}
.blockyright img {
    margin-top: 12px;
}
.blockyleft {
    display: inline-block;
    margin-left: 20px;
}
.blockydiv {
    width: 100%;
    height: 1px;
    background-color: #E9E9EF;
}
.blockyinfo {
    font-family: Roboto;
    font-size: 14px;
    color: #808292;
    margin: 30px;
}
.blockyinfo span {
    color: #253134;
    font-weight: 500;
    display: inline-block;
    border-bottom: 1px solid #D3DCEA;
    line-height: 20px;
    text-indent: 0px;
}
.block {
    background-color: #FFF;
    box-shadow: 0px 4px 30px rgba(22, 33, 74, 0.05);
	margin-right: 20px;
	border-radius: 5px;
	float: left;
}
.primary {
	border: solid 1px #91C6FF;
}
.primary-text {
	font-size: 11px;
	padding: 0px 4px;
	background-color: #4284CA;
	border-radius: 3px;
	color: #fff;
}
a {
	color: #4284CA;
}
</style>

</head>
<body>

<div class="blockin">
	<div class="blocktext">
		<p class="blocktitle">{{.Fanout}}</p>
		<p class="blockdesc"><a href="/fanout/{{.Fanout}}">/fanout/{{.Fanout}}</a></p>
	</div>
</div>

<div>
{{range $e := .Endpoints }}
	<div class="blockelem block {{if $e.Primary}}primary{{end}}">
		<div class="blockyleft"><p class="blockyname">{{$e.Name}}{{if $e.Primary}} <span class="primary-text">primary</span>{{end}}</p></div>
		<div class="blockydiv"></div>
		<div class="blockyinfo">
			<span>URL</span> {{$e.URL}}
			<br>
			<span>Method</span> {{$e.Method}}
			<br>
			<span>Timeout</span>{{if (gt $e.Timeout 0)}} {{$e.Timeout}}ms {{else}} default {{end}}
			
		</div>
	</div>
{{end}}
</div>

</body>
</html>
`
