// Étape 3 — Première application Express
const express = require('express');
const app = express();
app.use(express.json());

app.get('/', (req, res) => res.send('Bienvenue au restaurant'));
app.get('/health', (req, res) => res.json({ status: 'ok' }));

app.listen(process.env.PORT || 3000, () => console.log('API prête'));
