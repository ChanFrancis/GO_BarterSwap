package main

import "time"

// Structures de l'API, avec les tags JSON imposés par le sujet.

type User struct {
	ID            int       `json:"id"`
	Pseudo        string    `json:"pseudo"`
	Bio           string    `json:"bio,omitempty"`
	Ville         string    `json:"ville,omitempty"`
	Skills        []Skill   `json:"skills,omitempty"`
	CreditBalance int       `json:"credit_balance"`
	CreatedAt     time.Time `json:"created_at"`
}

type Skill struct {
	Nom    string `json:"nom"`    // ex : "Jardinage"
	Niveau string `json:"niveau"` // "débutant", "intermédiaire", "expert"
}

type Service struct {
	ID           int       `json:"id"`
	ProviderID   int       `json:"provider_id"`
	Titre        string    `json:"titre"`
	Description  string    `json:"description,omitempty"`
	Categorie    string    `json:"categorie"`
	DureeMinutes int       `json:"duree_minutes"` // durée estimée
	Credits      int       `json:"credits"`       // coût en crédits-temps
	Ville        string    `json:"ville,omitempty"`
	Actif        bool      `json:"actif"`
	CreatedAt    time.Time `json:"created_at"`
}

type Exchange struct {
	ID          int       `json:"id"`
	ServiceID   int       `json:"service_id"`
	RequesterID int       `json:"requester_id"` // celui qui demande
	OwnerID     int       `json:"owner_id"`     // celui qui propose
	Status      string    `json:"status"`       // pending, accepted, rejected, cancelled, completed
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreditTransaction struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	ExchangeID int       `json:"exchange_id"`
	Montant    int       `json:"montant"` // positif = crédit, négatif = débit
	Type       string    `json:"type"`    // "earn", "spend", "refund"
	CreatedAt  time.Time `json:"created_at"`
}

type Review struct {
	ID          int       `json:"id"`
	ExchangeID  int       `json:"exchange_id"`
	AuthorID    int       `json:"author_id"`
	TargetID    int       `json:"target_id"`
	Note        int       `json:"note"` // 1-5
	Commentaire string    `json:"commentaire,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserStats struct {
	UserID            int     `json:"user_id"`
	ServicesActifs    int     `json:"services_actifs"`
	EchangesCompletes int     `json:"echanges_completes"`
	CreditBalance     int     `json:"credit_balance"`
	NoteMoyenne       float64 `json:"note_moyenne"`
	NbAvis            int     `json:"nb_avis"`
	TotalGagne        int     `json:"total_gagne"`   // crédits gagnés au total
	TotalDepense      int     `json:"total_depense"` // crédits dépensés au total
}
