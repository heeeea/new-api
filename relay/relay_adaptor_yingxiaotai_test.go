package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	tasksora "github.com/QuantumNous/new-api/relay/channel/task/sora"
	"github.com/stretchr/testify/require"
)

func TestXaiUsesOpenAIVideoTaskAdaptor(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatform("48"))
	require.IsType(t, &tasksora.TaskAdaptor{}, adaptor)
}
