// Étape 2 — Une même règle métier, trois paradigmes
// Règle : total TTC = somme(prix * qte) * 1.2, arrondi à 2 décimales
// Exemple : [{prix:10, qte:2}, {prix:5, qte:1}] → 30
const arrondi = n => Math.round(n * 100) / 100;

// 1. Procédural : une boucle for et une variable qui s'accumule
function totalProcedural(plats) {
  // TODO
}

// 2. Orienté objet : une classe qui encapsule les plats
class Commande {
  constructor(plats) { this.plats = plats; }
  // TODO : méthode ajouter(plat) qui renvoie this (chaînable)
  // TODO : méthode totalTTC()
}

// 3. Fonctionnel : fonctions pures + map / reduce, aucune variable modifiée
const totalFonctionnel = plats => {
  // TODO
};

module.exports = { totalProcedural, Commande, totalFonctionnel };
