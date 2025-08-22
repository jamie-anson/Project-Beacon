package errors

import (
    stdErrors "errors"
    "testing"
)

func TestAppError_ConstructorsAndMethods(t *testing.T) {
    // New and Error formatting without cause
    e := New(ValidationError, "bad input")
    if e.Type != ValidationError || e.Message != "bad input" {
        t.Fatalf("unexpected New fields: %+v", e)
    }
    if e.Error() != "validation: bad input" {
        t.Fatalf("unexpected Error(): %q", e.Error())
    }

    // WithCode / WithDetails chaining
    e.WithCode("E123").WithDetails("missing field x")
    if e.Code != "E123" || e.Details != "missing field x" {
        t.Fatalf("WithCode/WithDetails failed: %+v", e)
    }

    // Newf
    nf := Newf(ConflictError, "resource %s exists", "abc")
    if nf.Type != ConflictError || nf.Message != "resource abc exists" {
        t.Fatalf("unexpected Newf: %+v", nf)
    }

    // Wrap and Wrapf include cause
    cause := stdErrors.New("boom")
    w := Wrap(cause, DatabaseError, "db op failed")
    if w.Cause == nil || w.Unwrap() != cause {
        t.Fatalf("Wrap did not set cause: %+v", w)
    }
    if w.Error() == "" || w.Type != DatabaseError {
        t.Fatalf("unexpected Wrap fields: %+v", w)
    }

    wf := Wrapf(cause, ExternalServiceError, "%s call failed", "ipfs")
    if wf.Type != ExternalServiceError || wf.Cause == nil {
        t.Fatalf("unexpected Wrapf: %+v", wf)
    }

    // Is on AppError compares Type+Code
    a := &AppError{Type: NotFoundError, Code: "X"}
    b := &AppError{Type: NotFoundError, Code: "X"}
    c := &AppError{Type: NotFoundError, Code: "Y"}
    if !a.Is(b) {
        t.Fatalf("expected a.Is(b) true")
    }
    if a.Is(c) {
        t.Fatalf("expected a.Is(c) false due to different code")
    }
}

func TestHelpers_IsType_GetType(t *testing.T) {
    base := NewTimeoutError("op1")
    if !IsType(base, TimeoutError) {
        t.Fatalf("IsType failed for base")
    }
    // wrapped error should report the OUTER type for IsType/GetType per implementation
    wrapped := Wrap(base, InternalError, "wrapped")
    if IsType(wrapped, TimeoutError) {
        t.Fatalf("IsType should not report inner type for wrapped error")
    }
    if GetType(wrapped) != InternalError {
        t.Fatalf("GetType should return outer type")
    }

    // non AppError defaults
    other := stdErrors.New("plain")
    if IsType(other, ValidationError) {
        t.Fatalf("plain error should not match AppError type")
    }
    if GetType(other) != InternalError {
        t.Fatalf("plain error GetType should be InternalError")
    }
}

func TestCommonConstructors(t *testing.T) {
    if NewValidationError("m").Type != ValidationError { t.Fatal("NewValidationError type") }
    if NewNotFoundError("thing").Type != NotFoundError { t.Fatal("NewNotFoundError type") }
    if NewConflictError("m").Type != ConflictError { t.Fatal("NewConflictError type") }
    if NewExternalServiceError("svc", stdErrors.New("x")).Type != ExternalServiceError { t.Fatal("NewExternalServiceError type") }
    if NewDatabaseError(stdErrors.New("x")).Type != DatabaseError { t.Fatal("NewDatabaseError type") }
    if NewCircuitBreakerError("svc").Type != CircuitBreakerError { t.Fatal("NewCircuitBreakerError type") }
    if NewTimeoutError("op").Type != TimeoutError { t.Fatal("NewTimeoutError type") }
    if NewInternalError("m").Type != InternalError { t.Fatal("NewInternalError type") }
}
