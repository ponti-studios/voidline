package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "voidline/internal/application/errors"
	"voidline/internal/application/validation"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// HandleError handles errors and returns appropriate HTTP responses
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	switch e := err.(type) {
	case apperrors.NotFoundError:
		c.JSON(http.StatusNotFound, ErrorResponse{Error: e.Error()})
	case validation.ValidationErrors:
		details := make(map[string]string)
		for _, v := range e {
			details[v.Field] = v.Message
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation failed",
			Details: details,
		})
	case apperrors.CannotDeleteError:
		c.JSON(http.StatusForbidden, ErrorResponse{Error: e.Error()})
	case apperrors.CascadeDeleteError:
		c.JSON(http.StatusConflict, ErrorResponse{Error: e.Error()})
	case apperrors.ReassignError:
		c.JSON(http.StatusConflict, ErrorResponse{Error: e.Error()})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}
}

// SuccessResponse represents a successful response with data
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// CreatedResponse returns a 201 Created response
func CreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// NoContentResponse returns a 204 No Content response
func NoContentResponse(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
