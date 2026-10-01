// Étape 1 — Consommer une API REST avec fetch (côté client)
const API = process.env.API_URL || 'http://localhost:4000';

async function getUser(id) {
  try {
    const res = await fetch(`${API}/users/${id}`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return await res.json();
  } catch (err) {
    console.log('Échec :', err.message);
    return null;
  }
}

async function creerPost(titre) {
  const res = await fetch(`${API}/posts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: titre, userId: 1 }),
  });
  const post = await res.json();
  console.log('Créé :', post.id, '(statut', res.status + ')');
}

(async () => {
  const u = await getUser(1);
  console.log(`${u.name} - ${u.email}`);
  await getUser(999);
  await creerPost('Mon premier POST');
})();
