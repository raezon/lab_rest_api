// Étape 1 — Consommer une API REST avec fetch (côté client)
// L'API de démonstration est lancée par apilab sur API_URL.
const API = process.env.API_URL || 'http://localhost:4000';

async function getUser(id) {
  // TODO 1 : appeler GET `${API}/users/${id}` avec fetch
  // TODO 2 : si la réponse n'est pas ok (res.ok), lever une Error(`HTTP ${res.status}`)
  // TODO 3 : renvoyer le JSON. En cas d'erreur, afficher 'Échec :' + message et renvoyer null
}

async function creerPost(titre) {
  // TODO 4 : envoyer un POST `${API}/posts` avec :
  //   - l'en-tête Content-Type: application/json
  //   - le corps JSON { title: titre, userId: 1 }
  // puis afficher : Créé : <id> (statut <code>)
}

(async () => {
  const u = await getUser(1);
  console.log(`${u.name} - ${u.email}`);
  await getUser(999);           // doit afficher : Échec : HTTP 404
  await creerPost('Mon premier POST');
})();
