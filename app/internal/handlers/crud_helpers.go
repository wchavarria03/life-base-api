package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// createForBike wraps a "bind JSON body, stamp the :id bike onto it, call
// service Create" endpoint. setBike copies the route's bike id into the
// input (each input type names that field BikeID).
func createForBike[TIn, TOut any](
	setBike func(input *TIn, bikeID string),
	fn func(ctx context.Context, input TIn) (TOut, error),
) gin.HandlerFunc {
	return func(c *gin.Context) {
		input, ok := bindJSON[TIn](c)
		if !ok {
			return
		}
		setBike(&input, c.Param("id"))
		item, err := fn(c.Request.Context(), input)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, item)
	}
}

// updateByParam wraps a "bind a partial-fields map, call service Update by
// the named route param" endpoint (updateHandler is the :id special case).
func updateByParam[TOut any](
	param string,
	fn func(ctx context.Context, id string, fields map[string]any) (TOut, error),
) gin.HandlerFunc {
	return func(c *gin.Context) {
		fields, ok := bindJSON[map[string]any](c)
		if !ok {
			return
		}
		item, err := fn(c.Request.Context(), c.Param(param), fields)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

// updateOrNotFound wraps "bind a request body, call a service update keyed
// by :id, 422 on error, 404 when the service reports nothing was updated".
func updateOrNotFound[TReq, TOut any](
	notFoundMsg string,
	fn func(ctx context.Context, id string, req TReq) (*TOut, error),
) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, ok := bindJSON[TReq](c)
		if !ok {
			return
		}
		item, err := fn(c.Request.Context(), c.Param("id"), req)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		if item == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": notFoundMsg})
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

// parseDateRange reads the required ?from= and ?to= (YYYY-MM-DD) query
// params, writing a 400 itself and returning ok=false when they are missing
// or malformed. fromStr/toStr are the raw values for callers that echo them.
func parseDateRange(c *gin.Context) (from, to time.Time, fromStr, toStr string, ok bool) {
	fromStr = c.Query("from")
	toStr = c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to are required"})
		return
	}
	var err error
	from, err = time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date, expected YYYY-MM-DD"})
		return
	}
	to, err = time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date, expected YYYY-MM-DD"})
		return
	}
	ok = true
	return
}
