// Package barterswap définit le domaine métier de l'application : les
// entités, les erreurs et les règles pures (validations, cycle de vie des
// échanges). Ce package ne dépend ni de HTTP ni de la base de données.
package barterswap

import "time"

// User est un membre de la plateforme, avec ses compétences et son solde de
// crédits-temps.
type User struct {
	ID            int       `json:"id"`
	Pseudo        string    `json:"pseudo"`
	Bio           string    `json:"bio,omitempty"`
	Ville         string    `json:"ville,omitempty"`
	Skills        []Skill   `json:"skills,omitempty"`
	CreditBalance int       `json:"credit_balance"`
	CreatedAt     time.Time `json:"created_at"`
}

// Skill est une compétence déclarée par un utilisateur.
type Skill struct {
	Nom    string `json:"nom"`    // ex : "Jardinage"
	Niveau string `json:"niveau"` // "débutant", "intermédiaire", "expert"
}

// Service est une annonce publiée par un utilisateur pour proposer une
// prestation, facturée en crédits-temps.
type Service struct {
	ID           int       `json:"id"`
	ProviderID   int       `json:"provider_id"`
	Titre        string    `json:"titre"`
	Description  string    `json:"description,omitempty"`
	Categorie    string    `json:"categorie"`
	DureeMinutes int       `json:"duree_minutes"`
	Credits      int       `json:"credits"`
	Ville        string    `json:"ville,omitempty"`
	Actif        bool      `json:"actif"`
	CreatedAt    time.Time `json:"created_at"`
}

// Exchange est une demande d'échange sur un service, avec son cycle de vie
// (pending, accepted, rejected, cancelled, completed).
type Exchange struct {
	ID          int       `json:"id"`
	ServiceID   int       `json:"service_id"`
	RequesterID int       `json:"requester_id"`
	OwnerID     int       `json:"owner_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreditTransaction est une entrée du journal des crédits-temps : le solde
// d'un utilisateur est la somme de ses transactions.
type CreditTransaction struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	ExchangeID int       `json:"exchange_id"`
	Montant    int       `json:"montant"` // positif = crédit, négatif = débit
	Type       string    `json:"type"`    // "earn", "spend", "refund"
	CreatedAt  time.Time `json:"created_at"`
}

// Review est un avis laissé après un échange terminé.
type Review struct {
	ID          int       `json:"id"`
	ExchangeID  int       `json:"exchange_id"`
	AuthorID    int       `json:"author_id"`
	TargetID    int       `json:"target_id"`
	Note        int       `json:"note"` // 1-5
	Commentaire string    `json:"commentaire,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserStats est le tableau de bord agrégé d'un utilisateur.
type UserStats struct {
	UserID            int     `json:"user_id"`
	ServicesActifs    int     `json:"services_actifs"`
	EchangesCompletes int     `json:"echanges_completes"`
	CreditBalance     int     `json:"credit_balance"`
	NoteMoyenne       float64 `json:"note_moyenne"`
	NbAvis            int     `json:"nb_avis"`
	TotalGagne        int     `json:"total_gagne"`
	TotalDepense      int     `json:"total_depense"`
}
