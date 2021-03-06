package echo

import (
	"github.com/v-lozhkin/deployProject/internal/pkg/image"
	"github.com/v-lozhkin/deployProject/internal/pkg/item"
	statpkg "github.com/v-lozhkin/deployProject/pkg/stat"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Stat struct {
	MethodDuration   *statpkg.Timer
	ExtensionCounter *prometheus.CounterVec
}

func (s Stat) collectors() []prometheus.Collector {
	return []prometheus.Collector{s.MethodDuration, s.ExtensionCounter}
}

type delivery struct {
	items  item.Usecase
	images image.Storage
	stat   Stat
}

func New(items item.Usecase, images image.Storage, stat promauto.Factory) item.EchoDelivery {
	ret := delivery{
		items:  items,
		images: images,
		stat: Stat{
			MethodDuration: &statpkg.Timer{HistogramVec: stat.NewHistogramVec(
				prometheus.HistogramOpts{Name: "echo_method_duration"},
				[]string{"method_name"},
			)},
			ExtensionCounter: stat.NewCounterVec(prometheus.CounterOpts{
				Name: "echo_uploader_extensions",
			}, []string{"extension"},
			)},
	}

	prometheus.MustRegister(ret.stat.collectors()...)

	return ret
}
