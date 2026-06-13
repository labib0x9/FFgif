package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

type Manager struct {
	globalMiddlewares []Middleware
}

func NewManager() *Manager {
	return &Manager{
		globalMiddlewares: make([]Middleware, 0),
	}
}

// Append middlewares to globalMiddlewares
func (m *Manager) Use(middlewares ...Middleware) {
	m.globalMiddlewares = append(m.globalMiddlewares, middlewares...)
}

// middlewares are wrapped as queue, First Come First Server (FCFS)
func (m *Manager) With(next http.Handler, middlewares ...Middleware) http.Handler {
	// for _, middle := range middlewares {
	// 	next = middle(next)
	// }
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}
	return next
}

func (m *Manager) WrapMux(next http.Handler) http.Handler {
	for i := len(m.globalMiddlewares) - 1; i >= 0; i-- {
		next = m.globalMiddlewares[i](next)
	}
	return next
}
