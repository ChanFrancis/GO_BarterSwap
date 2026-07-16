package barterswap

import "errors"

// Erreurs sentinelles du domaine. Le package api les traduit en codes HTTP
// via errors.Is / errors.As.
var (
	ErrIntrouvable         = errors.New("ressource introuvable")
	ErrInterdit            = errors.New("action réservée au propriétaire de la ressource")
	ErrCompetenceManquante = errors.New("vous ne possédez pas de compétence correspondant à cette catégorie")
	ErrServicePropre       = errors.New("impossible de demander un échange sur son propre service")
	ErrCreditsInsuffisants = errors.New("crédits insuffisants pour lancer cet échange")
	ErrDejaReserve         = errors.New("ce service a déjà un échange en cours")
	ErrTransitionInvalide  = errors.New("cette action n'est pas possible dans l'état actuel de l'échange")
	ErrEchangeNonTermine   = errors.New("on ne peut noter qu'un échange terminé")
	ErrDejaNote            = errors.New("vous avez déjà noté cet échange")
)

// ValidationError signale une entrée utilisateur invalide (traduite en
// HTTP 400).
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }
