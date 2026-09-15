package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
)

func (a *API) billingAccounts(w http.ResponseWriter, r *http.Request) {
	if !a.requireSystemPermission(w, r) {
		return
	}
	if r.Method == http.MethodGet {
		accounts, err := a.repo.ListBillingAccounts(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		for i := range accounts {
			subscription, err := a.repo.GetSubscription(r.Context(), accounts[i].ID)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				fail(w, http.StatusInternalServerError, err)
				return
			}
			accounts[i].SubscriptionStatus = subscription.Status
		}
		respond(w, http.StatusOK, accounts)
		return
	}
	var account BillingAccount
	if !decode(w, r, &account) {
		return
	}
	if err := validBillingAccount(account); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	if err := a.repo.CreateBillingAccount(r.Context(), &account); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusCreated, account)
}

func (a *API) billingAccount(w http.ResponseWriter, r *http.Request) {
	if !a.requireSystemPermission(w, r) {
		return
	}
	id := r.PathValue("id")
	if r.Method == http.MethodGet {
		account, err := a.repo.GetBillingAccount(r.Context(), id)
		if notFound(w, err) {
			return
		}
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		respond(w, http.StatusOK, account)
		return
	}
	var account BillingAccount
	if !decode(w, r, &account) {
		return
	}
	if err := validBillingAccount(account); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	account.ID = id
	if err := a.repo.UpdateBillingAccount(r.Context(), &account); notFound(w, err) {
		return
	} else if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, account)
}

func (a *API) billingSubscription(w http.ResponseWriter, r *http.Request) {
	if !a.requireSystemPermission(w, r) {
		return
	}
	billingAccountID := r.PathValue("id")
	if r.Method == http.MethodGet {
		subscription, err := a.repo.GetSubscription(r.Context(), billingAccountID)
		if notFound(w, err) {
			return
		}
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		respond(w, http.StatusOK, subscription)
		return
	}
	if _, err := a.repo.GetBillingAccount(r.Context(), billingAccountID); notFound(w, err) {
		return
	} else if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	var subscription Subscription
	if !decode(w, r, &subscription) {
		return
	}
	subscription.BillingAccountID = billingAccountID
	if err := validSubscription(subscription); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	if err := a.repo.UpsertSubscription(r.Context(), &subscription); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, subscription)
}

func validBillingAccount(account BillingAccount) error {
	if strings.TrimSpace(account.Name) == "" {
		return errors.New("billing account name is required")
	}
	return nil
}

func validSubscription(subscription Subscription) error {
	if strings.TrimSpace(subscription.PlanCode) == "" {
		return errors.New("planCode is required")
	}
	switch subscription.Status {
	case "trialing", "active", "past_due", "canceled":
		return nil
	default:
		return errors.New("status must be trialing, active, past_due, or canceled")
	}
}
