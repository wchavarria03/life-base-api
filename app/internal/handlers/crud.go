package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// listHandler wraps a "list everything" service call — the shape repeated,
// nearly verbatim, across most List methods in this package. fn is usually
// a bound method value like h.svc.List.
func listHandler[T any](fn func(ctx context.Context) (T, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := fn(c.Request.Context())
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, items)
	}
}

// createHandler wraps a "bind JSON body, call service Create" endpoint.
func createHandler[TIn, TOut any](fn func(ctx context.Context, input TIn) (TOut, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		input, ok := bindJSON[TIn](c)
		if !ok {
			return
		}
		item, err := fn(c.Request.Context(), input)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, item)
	}
}

// updateHandler wraps a "bind a partial-fields map, call service Update by
// :id" endpoint.
func updateHandler[TOut any](fn func(ctx context.Context, id string, fields map[string]any) (TOut, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		fields, ok := bindJSON[map[string]any](c)
		if !ok {
			return
		}
		item, err := fn(c.Request.Context(), c.Param("id"), fields)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

// deleteHandler wraps a "call service Delete by :id, return 204" endpoint.
func deleteHandler(fn func(ctx context.Context, id string) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := fn(c.Request.Context(), c.Param("id")); err != nil {
			internalError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
