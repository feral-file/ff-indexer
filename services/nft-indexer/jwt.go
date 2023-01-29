package main

import "github.com/dgrijalva/jwt-go"

type SubscriptionPlan string

const (
	SubscriptionNone    = SubscriptionPlan("none")
	SubscriptionPremium = SubscriptionPlan("autonomy-premium")
)

type PlanJWTClaim struct {
	jwt.StandardClaims
	Plan SubscriptionPlan `json:"plan"`
}
