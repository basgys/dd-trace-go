// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package httpsec

import (
	"github.com/basgys/dd-trace-go/internal/appsec"
	"github.com/basgys/dd-trace-go/internal/appsec/listener/httpsec"
)

func init() {
	appsec.AddWAFEventListener(httpsec.Install)
}
