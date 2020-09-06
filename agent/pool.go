package agent

import (
	"sync"
)

type AgentPool struct {
	pool *sync.Pool
}

func NewAgentPool() *AgentPool {
	return &AgentPool{
		pool: &sync.Pool{
			New: func() interface{} {
				agent, err := NewAgent()
				if err != nil {
					panic(err)
				}

				return agent
			},
		},
	}
}

func (p *AgentPool) Get() *Agent {
	return p.pool.Get().(*Agent)
}

func (p *AgentPool) Put(agent *Agent) {
	p.pool.Put(agent)
}
