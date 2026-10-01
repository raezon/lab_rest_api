// Étape 2 — Une même règle métier, trois paradigmes
// Règle : total TTC = somme(prix * qte) * 1.2, arrondi à 2 décimales
const arrondi = n => Math.round(n * 100) / 100;

// 1. Procédural
function totalProcedural(plats) {
  let total = 0;
  for (let i = 0; i < plats.length; i++) {
    total += plats[i].prix * plats[i].qte;
  }
  return arrondi(total * 1.2);
}

// 2. Orienté objet
class Commande {
  constructor(plats) { this.plats = plats; }
  ajouter(plat) { this.plats.push(plat); return this; }
  totalTTC() { return arrondi(this.plats.reduce((t, p) => t + p.prix * p.qte, 0) * 1.2); }
}

// 3. Fonctionnel (fonctions pures, composition)
const sousTotal = p => p.prix * p.qte;
const somme = (a, b) => a + b;
const totalFonctionnel = plats => arrondi(plats.map(sousTotal).reduce(somme, 0) * 1.2);

module.exports = { totalProcedural, Commande, totalFonctionnel };
