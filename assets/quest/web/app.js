'use strict';
// app.js — interface de la quête. Aucune dépendance : tout tient dans ce fichier.

const $ = (selecteur) => document.querySelector(selecteur);
const esc = (s) => String(s).replace(/[&<>"]/g, (ch) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[ch]);

async function api(chemin, corps) {
  const options = corps === undefined ? {} : { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(corps) };
  const r = await fetch('/api/' + chemin, options);
  if (!r.ok) throw new Error((await r.text()) || r.statusText);
  return r.json();
}

// ---------- coloration syntaxique ----------

const JETONS = /(\/\/[^\n]*|\/\*[\s\S]*?\*\/)|(`(?:\\[\s\S]|[^`\\])*`|"(?:\\.|[^"\\\n])*"|'(?:\\.|[^'\\\n])*')|\b(\d+(?:\.\d+)?)\b|\b(const|let|var|function|return|if|else|for|while|of|in|new|class|extends|async|await|try|catch|finally|throw|typeof|instanceof|switch|case|break|default|do|continue|delete|true|false|null|undefined|this|module|require)\b|([A-Za-z_$][\w$]*)(?=\s*\()|(=>|\.\.\.)/g;
const CLASSES = ['c', 's', 'n', 'k', 'f', 'o'];

function colorer(code) {
  let html = '';
  let dernier = 0;
  for (const m of code.matchAll(JETONS)) {
    html += esc(code.slice(dernier, m.index));
    const type = CLASSES[m.slice(1).findIndex((g) => g !== undefined)];
    html += `<span class="t-${type}">${esc(m[0])}</span>`;
    dernier = m.index + m[0].length;
  }
  return html + esc(code.slice(dernier));
}

// ---------- mini-markdown des cours ----------

function enLigne(texte) {
  return texte
    .split(/(`[^`]+`)/)
    .map((p) =>
      p.length > 2 && p.startsWith('`') && p.endsWith('`')
        ? `<code>${esc(p.slice(1, -1))}</code>`
        : esc(p)
            .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
            .replace(/\*([^*\s][^*]*)\*/g, '<em>$1</em>')
    )
    .join('');
}

function md(src) {
  const lignes = src.split('\n');
  let html = '';
  let i = 0;
  const tantQue = (test, debut) => {
    const bloc = [];
    while (i < lignes.length && test(lignes[i])) bloc.push(lignes[i++].replace(debut, ''));
    return bloc;
  };
  while (i < lignes.length) {
    const l = lignes[i];
    if (l.startsWith('```')) {
      i++;
      const code = tantQue((x) => !x.startsWith('```'));
      i++;
      html += `<pre class="code"><code>${colorer(code.join('\n'))}</code></pre>`;
    } else if (/^#{2,3} /.test(l)) {
      const n = l.startsWith('### ') ? 3 : 2;
      html += `<h${n}>${enLigne(l.replace(/^#+ /, ''))}</h${n}>`;
      i++;
    } else if (l.startsWith('> ')) {
      html += `<blockquote><p>${enLigne(tantQue((x) => x.startsWith('> '), /^> /).join(' '))}</p></blockquote>`;
    } else if (/^\s*- /.test(l)) {
      html += '<ul>' + tantQue((x) => /^\s*- /.test(x), /^\s*- /).map((x) => `<li>${enLigne(x)}</li>`).join('') + '</ul>';
    } else if (/^\d+\. /.test(l)) {
      // une liste numérotée peut contenir des sous-lignes indentées
      let items = '';
      while (i < lignes.length && /^\d+\. /.test(lignes[i])) {
        let item = enLigne(lignes[i++].replace(/^\d+\. /, ''));
        const sous = tantQue((x) => /^\s+- /.test(x), /^\s+- /);
        if (sous.length) item += '<ul>' + sous.map((x) => `<li>${enLigne(x)}</li>`).join('') + '</ul>';
        items += `<li>${item}</li>`;
      }
      html += `<ol>${items}</ol>`;
    } else if (l.trim() === '') {
      i++;
    } else {
      html += `<p>${enLigne(tantQue((x) => x.trim() !== '' && !/^(```|#{2,3} |> |\s*- |\d+\. )/.test(x)).join(' '))}</p>`;
    }
  }
  return html;
}

// ---------- éditeur : un <textarea> transparent posé sur le code coloré ----------

class Editeur {
  constructor(racine, surSaisie) {
    this.ta = racine.querySelector('textarea');
    this.code = racine.querySelector('.rendu code');
    this.gouttiere = racine.querySelector('.gouttiere');
    this.nbLignes = 0;
    this.echap = false;
    this.ta.addEventListener('input', () => {
      this.peindre();
      surSaisie(this.ta.value);
    });
    this.ta.addEventListener('scroll', () => this.defiler());
    this.ta.addEventListener('keydown', (e) => this.touche(e));
  }

  get valeur() {
    return this.ta.value;
  }

  set valeur(v) {
    this.ta.value = v;
    this.ta.scrollTop = this.ta.scrollLeft = 0;
    this.peindre();
    this.defiler();
  }

  peindre() {
    const v = this.ta.value;
    this.code.innerHTML = colorer(v) + '\n';
    const n = v.split('\n').length;
    if (n !== this.nbLignes) {
      this.nbLignes = n;
      this.gouttiere.textContent = Array.from({ length: n }, (_, k) => k + 1).join('\n');
    }
  }

  defiler() {
    this.code.style.transform = `translate(${-this.ta.scrollLeft}px, ${-this.ta.scrollTop}px)`;
    this.gouttiere.scrollTop = this.ta.scrollTop;
  }

  // insertText conserve l'historique d'annulation (Ctrl+Z) ; setRangeText sert de repli.
  inserer(texte) {
    const commande = texte === '' ? 'delete' : 'insertText';
    if (!document.execCommand(commande, false, texte)) {
      this.ta.setRangeText(texte, this.ta.selectionStart, this.ta.selectionEnd, 'end');
      this.ta.dispatchEvent(new Event('input'));
    }
  }

  touche(e) {
    const ta = this.ta;
    const d = ta.selectionStart;
    const f = ta.selectionEnd;
    const v = ta.value;
    const echap = this.echap;
    this.echap = e.key === 'Escape';
    if (e.ctrlKey || e.metaKey || e.altKey) return;

    if (e.key === 'Tab') {
      if (echap) return; // Échap puis Tab : sortir de l'éditeur au clavier
      e.preventDefault();
      if (!e.shiftKey) return this.inserer('  ');
      const debut = v.lastIndexOf('\n', d - 1) + 1;
      if (v.startsWith('  ', debut)) {
        ta.setSelectionRange(debut, debut + 2);
        this.inserer('');
        const pos = Math.max(debut, d - 2);
        ta.setSelectionRange(pos, pos);
      }
    } else if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      const avant = v.slice(v.lastIndexOf('\n', d - 1) + 1, d);
      const retrait = avant.match(/^[ \t]*/)[0];
      const ouvre = /([{(\[]|=>)\s*$/.test(avant);
      if (ouvre && /^[})\]]/.test(v.slice(f))) {
        this.inserer('\n' + retrait + '  \n' + retrait);
        const pos = ta.selectionStart - retrait.length - 1;
        ta.setSelectionRange(pos, pos);
      } else {
        this.inserer('\n' + retrait + (ouvre ? '  ' : ''));
      }
    } else if (d === f && '([{'.includes(e.key) && /^(\s|[)\]}]|$)/.test(v.slice(d, d + 1))) {
      e.preventDefault();
      this.inserer(e.key + ')]}'['([{'.indexOf(e.key)]);
      ta.setSelectionRange(d + 1, d + 1);
    } else if (d === f && ')]}'.includes(e.key) && v[d] === e.key) {
      e.preventDefault();
      ta.setSelectionRange(d + 1, d + 1);
    }
  }
}

// ---------- état ----------

const E = {
  quetes: [],
  av: {},
  score: null,
  node: '',
  dossier: '',
  id: null,
  vue: 'quete', // quete | cours | memo | score
  onglet: 'cours', // cours | defi | doc | indices
  fichier: 'exemple', // exemple | defi
  panneau: 'sortie', // sortie | requete | tests
  sortie: null,
  resultats: null,
  occupe: false,
  parcours: { titre: '', sousTitre: '' },
  profil: null, // { prenom, nom, ecole } une fois l'élève inscrit
  req: { methode: 'GET', chemin: '/', corps: '', entetes: '' }, // formulaire de l'onglet Requête
  reponse: null, // dernière réponse du serveur de l'élève
  jeton: '', // dernier token reçu : remplace {{token}}
};

let editeur;

const quete = () => E.quetes.find((q) => q.id === E.id);
const rang = (id) => E.quetes.findIndex((q) => q.id === id);
const debloquee = (i) => i <= 0 || E.av[E.quetes[i - 1].id].fait || E.av[E.quetes[i - 1].id].solutionVue;
const enServeur = () => quete().serveur && E.fichier === 'defi'; // le défi ouvert est un serveur HTTP
const premiereAFaire = () => Math.max(E.quetes.findIndex((q) => !E.av[q.id].fait), 0);

function memoriser() {
  try {
    localStorage.setItem('brigade', JSON.stringify({ id: E.id, vue: E.vue }));
  } catch {}
}

function souvenir() {
  try {
    return JSON.parse(localStorage.getItem('brigade')) || {};
  } catch {
    return {};
  }
}

// ---------- thème clair / sombre ----------

function rendreTheme() {
  const clair = document.documentElement.dataset.theme === 'light';
  const b = $('#btn-theme');
  // Le bouton annonce le thème vers lequel il bascule.
  b.innerHTML = clair ? '<span aria-hidden="true">🌙</span> <span class="lib-theme">Sombre</span>' : '<span aria-hidden="true">☀️</span> <span class="lib-theme">Clair</span>';
  b.title = clair ? 'Passer au thème sombre' : 'Passer au thème clair';
  b.setAttribute('aria-label', b.title);
}

function basculerTheme() {
  const theme = document.documentElement.dataset.theme === 'light' ? 'dark' : 'light';
  document.documentElement.dataset.theme = theme;
  try {
    localStorage.setItem('brigade-theme', theme);
  } catch {}
  rendreTheme();
}

// ---------- inscription ----------

function ouvrirInscription() {
  const f = $('#form-inscription');
  for (const champ of ['prenom', 'nom', 'ecole']) f.elements[champ].value = E.profil ? E.profil[champ] : '';
  // À la première visite, l'inscription est obligatoire ; ensuite c'est une simple correction.
  $('#annuler-inscription').hidden = !E.profil;
  $('#titre-inscription').textContent = E.profil ? 'Modifier mon inscription' : 'Bienvenue en cuisine 👋';
  f.querySelector('[type=submit]').textContent = E.profil ? 'Enregistrer' : 'Commencer';
  $('#erreur-inscription').hidden = true;
  $('#inscription').hidden = false;
  f.elements.prenom.focus();
}

async function validerInscription(e) {
  e.preventDefault();
  const f = e.target;
  try {
    const r = await api('profil', { prenom: f.elements.prenom.value, nom: f.elements.nom.value, ecole: f.elements.ecole.value });
    E.profil = r.profil;
    $('#inscription').hidden = true;
    rendre();
  } catch (err) {
    $('#erreur-inscription').textContent = err.message;
    $('#erreur-inscription').hidden = false;
  }
}

// ---------- sauvegarde automatique ----------

let minuterie = null;
let aSauver = null;

function surSaisie(valeur) {
  E.av[E.id][E.fichier === 'exemple' ? 'exemple' : 'code'] = valeur;
  aSauver = { id: E.id, fichier: E.fichier, code: valeur };
  clearTimeout(minuterie);
  minuterie = setTimeout(sauver, 800);
}

function sauver() {
  clearTimeout(minuterie);
  if (!aSauver) return;
  const envoi = aSauver;
  aSauver = null;
  api('sauver', envoi).catch(() => {});
}

// ---------- rendu ----------

const etoiles = (n) => `<span class="etoiles" title="${n} étoile(s) sur 3">${'★'.repeat(n)}${'☆'.repeat(3 - n)}</span>`;
const jauge = (part) => `<span class="jauge"><i style="width:${Math.min(100, Math.round(part * 100))}%"></i></span>`;

function onglets(cible, liste, actif) {
  $(cible).innerHTML = liste
    .map(([cle, libelle]) => `<button role="tab" data-cle="${cle}" aria-selected="${cle === actif}">${libelle}</button>`)
    .join('');
}

function rendreBarres() {
  const s = E.score;
  document.title = E.parcours.titre;
  $('#marque').innerHTML = `<span aria-hidden="true">🍽</span> ${esc(E.parcours.titre)} <small>${esc(E.parcours.sousTitre)}</small>`;
  $('#score-haut').innerHTML =
    (E.profil ? `<span class="eleve">${esc(E.profil.prenom)} ${esc(E.profil.nom.slice(0, 1))}.</span>` : '') +
    `<span class="rang">${esc(s.rang)}</span>${jauge(s.xp / s.xpMax)}<span>${s.xp} XP</span><span class="etoiles">★ ${s.etoiles}</span>`;
  for (const b of document.querySelectorAll('#barre-activite button')) b.classList.toggle('actif', b.dataset.vue === E.vue);
  $('#barre-etat').innerHTML =
    `<span>Node ${esc(E.node)}</span><span>Quêtes ${s.faites}/${s.total}</span>` +
    `<span>Ctrl+Entrée : exécuter · Ctrl+S : vérifier</span><span class="droite">Code enregistré dans ${esc(E.dossier)}</span>`;
}

function rendreLateral() {
  let html = '';
  let chapitre = null;
  E.quetes.forEach((q, i) => {
    if (q.chapitre !== chapitre) {
      if (chapitre !== null) html += '</ul>';
      chapitre = q.chapitre;
      const duChapitre = E.quetes.filter((x) => x.chapitre === chapitre);
      html += `<h2><span>${esc(chapitre)}</span><span>${duChapitre.filter((x) => E.av[x.id].fait).length}/${duChapitre.length}</span></h2><ul>`;
    }
    const a = E.av[q.id];
    const ouverte = debloquee(i);
    const accessible = ouverte || E.vue === 'cours'; // les cours se lisent librement
    const classes = [q.id === E.id ? 'courante' : '', a.fait ? 'faite' : ''].join(' ');
    html +=
      `<li><button data-id="${q.id}" class="${classes}" ${accessible ? '' : 'disabled title="Termine la quête précédente pour débloquer celle-ci"'}>` +
      `<span class="ic" aria-hidden="true">${a.fait ? '✔' : ouverte ? '○' : '🔒'}</span>` +
      `<span class="nom">${esc(q.titre)}</span>` +
      `<span class="et">${a.fait ? '★'.repeat(a.etoiles) : q.xp + ' XP'}</span></button></li>`;
  });
  $('#lateral').innerHTML = html + '</ul>';
}

function rendreConsignes() {
  const q = quete();
  const a = E.av[q.id];
  const i = rang(q.id);
  const restants = q.nbIndices - a.indicesReveles.length;
  onglets('#onglets-consignes', [
    ['cours', '📖 Cours'],
    ['defi', '🎯 Défi' + (a.fait ? ' <span class="pastille ok">✔</span>' : '')],
    ['doc', '📑 Doc'],
    ['indices', '💡 Indices' + (restants > 0 && !a.fait ? ` <span class="pastille">${restants}</span>` : '')],
  ], E.onglet);

  const entete = `<p class="surtitre">${esc(q.chapitre)} · quête ${i + 1}/${E.quetes.length}</p><h1>${esc(q.titre)}</h1><p class="notion">${esc(q.notion)}</p>`;
  let html = '';
  if (E.onglet === 'cours') {
    html = entete + md(q.cours) +
      '<hr><p><strong>Étape suivante :</strong> teste l\'exemple, puis relève le défi.</p>' +
      '<button class="bouton" data-action="exemple">▶ Tester l\'exemple</button>' +
      '<button class="bouton primaire" data-action="defi">🎯 Passer au défi</button>';
  } else if (E.onglet === 'defi') {
    html = entete + md(q.consigne);
    if (a.fait) {
      const suivante = E.quetes[i + 1];
      html +=
        `<div class="encart ok"><strong>Quête validée</strong> ${etoiles(a.etoiles)} · ${a.xp} XP<br>💡 À retenir : ${enLigne(q.retenir)}</div>` +
        (suivante
          ? `<button class="bouton primaire" data-action="suivante">Quête suivante : ${esc(suivante.titre)} →</button>`
          : '<div class="encart ok">🏆 Parcours terminé. Bravo, chef !</div><button class="bouton primaire" data-action="score">Voir le tableau des scores</button>');
    } else {
      html += `<div class="encart">À gagner : <strong>${q.xp} XP</strong> (+20 % si tu réussis du premier coup, −10 % par indice utilisé).</div>`;
    }
  } else if (E.onglet === 'doc') {
    html = entete + md(q.doc);
  } else {
    html = entete;
    html += a.indicesReveles.length
      ? a.indicesReveles.map((ind, k) => `<div class="encart"><strong>Indice ${k + 1}</strong><br>${enLigne(ind)}</div>`).join('')
      : '<p class="muet">Aucun indice révélé. Relis le cours et la doc avant d\'en demander un : chaque indice coûte 10 % des XP.</p>';
    if (!a.fait && restants > 0) html += `<button class="bouton" data-action="indice">💡 Révéler un indice (reste ${restants})</button>`;
    if (a.solution) {
      html += `<h2>Correction</h2><pre class="code"><code>${colorer(a.solution)}</code></pre>`;
    } else {
      html += '<button class="bouton danger" data-action="solution">Voir la correction (XP divisés par 4)</button>';
    }
  }
  const zone = $('#contenu-consignes');
  zone.innerHTML = html;
  zone.scrollTop = 0;
}

function rendreAtelier() {
  onglets('#onglets-fichiers', [['exemple', 'exemple.js'], ['defi', 'defi.js']], E.fichier);
  $('#btn-verifier').hidden = E.fichier !== 'defi';
  // Un serveur ne « s'exécute » pas comme un script : on lui envoie une requête.
  $('#btn-executer').textContent = enServeur() ? '▶ Envoyer la requête' : '▶ Exécuter';
  for (const b of ['#btn-executer', '#btn-verifier', '#btn-reinit']) $(b).disabled = E.occupe;
  rendrePanneau();
}

function rendrePanneau() {
  const res = E.resultats;
  const reussis = res ? res.filter((r) => r.ok).length : 0;
  onglets('#onglets-panneau', [
    ['sortie', 'Sortie'],
    ...(quete().serveur ? [['requete', 'Requête']] : []),
    ['tests', 'Tests' + (res ? ` <span class="pastille ${reussis === res.length ? 'ok' : 'ko'}">${reussis}/${res.length}</span>` : '')],
  ], E.panneau);
  let html;
  if (E.occupe) {
    html = '<p class="muet">Exécution en cours…</p>';
  } else if (E.panneau === 'requete') {
    html = panneauRequete();
  } else if (E.panneau === 'sortie') {
    html = E.sortie === null
      ? '<pre class="sortie vide">Clique sur « Exécuter » pour lancer le fichier avec Node. Ce que tu écris avec console.log s\'affiche ici.</pre>'
      : `<pre class="sortie">${esc(E.sortie) || '<span class="muet">(aucune sortie : ajoute un console.log pour voir une valeur)</span>'}</pre>`;
  } else if (!res) {
    html = '<p class="muet">Ouvre defi.js puis clique sur « Vérifier » pour lancer les tests de la quête.</p>';
  } else {
    const tout = reussis === res.length;
    html =
      `<p class="bilan ${tout ? 'ok' : 'ko'}">${tout ? '🎉 ' : ''}${reussis}/${res.length} tests réussis${tout ? ' — quête validée !' : '. Corrige et relance.'}</p>` +
      '<ul class="tests">' +
      res.map((r) => `<li class="${r.ok ? 'ok' : 'ko'}"><span aria-hidden="true">${r.ok ? '✔' : '✖'}</span><div><strong>${esc(r.nom)}</strong>${r.detail ? `<pre>${esc(r.detail)}</pre>` : ''}</div></li>`).join('') +
      '</ul>';
  }
  $('#contenu-panneau').innerHTML = html;
  // L'onglet Requête montre un formulaire et une réponse : on lui donne plus de hauteur.
  $('#panneau').classList.toggle('grand', E.panneau === 'requete');
}

// Onglet « Requête » : un mini-client HTTP branché sur le serveur de l'élève.
function panneauRequete() {
  const q = quete();
  const r = E.req;
  const puces = q.requetes
    .map((x, i) => `<button class="puce" data-req="${i}" title="Remplir le formulaire avec cette requête"><b>${esc(x.methode)}</b> ${esc(x.chemin)}${x.corps ? ' <i>+ corps</i>' : ''}${x.entetes ? ' <i>+ en-tête</i>' : ''}</button>`)
    .join('');
  let html =
    `<div class="puces">${puces}</div>` +
    '<div class="requete">' +
    `<select data-champ="methode" aria-label="Méthode">${['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((m) => `<option${m === r.methode ? ' selected' : ''}>${m}</option>`).join('')}</select>` +
    `<input data-champ="chemin" aria-label="Chemin" spellcheck="false" value="${esc(r.chemin)}">` +
    '<button class="bouton primaire" data-envoi="1">Envoyer</button></div>' +
    '<div class="requete-plus">' +
    `<label>Corps JSON<textarea data-champ="corps" rows="1" spellcheck="false" placeholder='{"nom":"Burger","prix":11}'>${esc(r.corps)}</textarea></label>` +
    `<label>En-têtes (un par ligne)<textarea data-champ="entetes" rows="1" spellcheck="false" placeholder="Authorization: Bearer {{token}}">${esc(r.entetes)}</textarea></label>` +
    '</div>';
  const rep = E.reponse;
  if (!rep) {
    return html + '<p class="muet">Choisis une requête ci-dessus ou écris la tienne, puis « Envoyer ». Ton serveur (defi.js) est démarré automatiquement et garde ses données tant que tu ne modifies pas le code.</p>';
  }
  if (rep.redemarre) html += '<p class="muet">↻ Serveur (re)démarré avec ton code actuel : ses données en mémoire repartent de zéro.</p>';
  if (rep.erreur) {
    html += `<p class="statut s5">Échec</p><pre class="sortie">${esc(rep.erreur)}</pre>`;
  } else {
    const x = rep.reponse;
    let corps = x.corps;
    try {
      corps = JSON.stringify(JSON.parse(x.corps), null, 2);
    } catch {}
    html +=
      `<p class="statut s${Math.floor(x.statut / 100)}">${x.statut} ${esc(x.libelle)} <span class="muet">· ${x.dureeMs} ms</span></p>` +
      (x.flux ? '<p class="muet">Flux continu (SSE) : connexion coupée après 1,5 s, voici ce qui a été reçu.</p>' : '') +
      `<details class="entetes"><summary>${x.entetes.length} en-têtes de réponse</summary><pre class="sortie">${x.entetes.map(([k, v]) => `${esc(k)}: ${esc(v)}`).join('\n')}</pre></details>` +
      `<pre class="sortie">${corps ? colorer(corps) : '<span class="muet">(corps vide)</span>'}</pre>`;
  }
  if (rep.console) html += `<p class="muet console-titre">Console du serveur</p><pre class="sortie muet">${esc(rep.console)}</pre>`;
  return html;
}

function pageCours() {
  const q = quete();
  const ouverte = debloquee(rang(q.id));
  return (
    `<p class="surtitre">${esc(q.chapitre)}</p><h1>${esc(q.titre)}</h1><p class="notion">${esc(q.notion)}</p>` +
    md(q.cours) +
    `<h2>Exemple à tester</h2><pre class="code"><code>${colorer(E.av[q.id].exemple)}</code></pre>` +
    `<h2>Mémo</h2>${md(q.doc)}<hr>` +
    (ouverte
      ? '<button class="bouton primaire" data-action="ouvrir">Ouvrir cette quête dans l\'éditeur →</button>'
      : '<div class="encart attention">🔒 Le défi de cette quête se débloque quand la précédente est terminée. Le cours, lui, reste en lecture libre.</div>')
  );
}

function pageMemo() {
  let html = `<h1>📑 Mémo — ${esc(E.parcours.titre)}</h1>` + '<p class="notion">Toute la documentation des quêtes, réunie pour la retrouver rapidement.</p>';
  let chapitre = null;
  for (const q of E.quetes) {
    if (q.chapitre !== chapitre) {
      chapitre = q.chapitre;
      html += `<hr><p class="surtitre">Chapitre ${esc(chapitre)}</p>`;
    }
    html += `<h2>${esc(q.titre)} — ${esc(q.notion)}</h2>` + md(q.doc).replace(/<(\/?)h2>/g, '<$1h3>');
  }
  return html;
}

function pageScore() {
  const s = E.score;
  const p = E.profil;
  const termine = s.faites === s.total;
  const aujourdhui = new Date().toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' });
  let html =
    `<p class="surtitre">${esc(E.parcours.titre)} · ${esc(E.parcours.sousTitre)}</p>` +
    `<h1>🏆 ${termine ? 'Bilan final' : 'Tableau des scores'}</h1>` +
    (p
      ? `<p class="identite"><strong>${esc(p.prenom)} ${esc(p.nom)}</strong> · ${esc(p.ecole)} · le ${aujourdhui} ` +
        '<button class="bouton sans-impression" data-action="profil">Modifier</button></p>'
      : '') +
    (termine
      ? `<div class="encart ok"><strong>Parcours terminé, bravo${p ? ' ' + esc(p.prenom) : ''} !</strong> Les ${s.total} quêtes sont validées. ` +
        `Score final : <strong>${s.xp} XP sur ${s.xpMax}</strong> (${Math.round((s.xp / s.xpMax) * 100)} %), ${s.etoiles} étoiles sur ${s.etoilesMax}, rang ${esc(s.rang)}.</div>`
      : `<p class="notion">Il reste ${s.total - s.faites} quête${s.total - s.faites > 1 ? 's' : ''} avant le bilan final.</p>`) +
    '<div class="cartes">' +
    `<div class="carte"><div class="val">${esc(s.rang)}</div><div class="lib">${s.suivant ? 'Prochain rang : ' + esc(s.suivant) : 'Rang maximal atteint'}</div></div>` +
    `<div class="carte"><div class="val">${s.xp} XP</div><div class="lib">sur ${s.xpMax} possibles · ${Math.round((s.xp / s.xpMax) * 100)} %</div>${jauge(s.xp / s.xpMax)}</div>` +
    `<div class="carte"><div class="val">${s.faites}/${s.total}</div><div class="lib">quêtes validées</div>${jauge(s.faites / s.total)}</div>` +
    `<div class="carte"><div class="val"><span class="etoiles">★</span> ${s.etoiles}/${s.etoilesMax}</div><div class="lib">étoiles</div>${jauge(s.etoiles / s.etoilesMax)}</div>` +
    '</div><h2>Par chapitre</h2>';
  for (const chapitre of [...new Set(E.quetes.map((q) => q.chapitre))]) {
    const liste = E.quetes.filter((q) => q.chapitre === chapitre);
    const faites = liste.filter((q) => E.av[q.id].fait).length;
    html += `<div class="chapitre-ligne"><span>${esc(chapitre)}</span>${jauge(faites / liste.length)}<span>${faites}/${liste.length}</span></div>`;
  }
  html += '<h2>Détail des quêtes</h2><table class="scores"><thead><tr><th>Quête</th><th>Étoiles</th><th class="num">XP</th><th class="num">Essais</th><th class="num">Indices</th></tr></thead><tbody>';
  E.quetes.forEach((q, i) => {
    const a = E.av[q.id];
    html +=
      `<tr><td>${i + 1}. ${esc(q.titre)}</td><td>${a.fait ? etoiles(a.etoiles) : '<span class="muet">—</span>'}</td>` +
      `<td class="num">${a.fait ? a.xp : 0} / ${q.xp}</td><td class="num">${a.essais}</td><td class="num">${a.indices}</td></tr>`;
  });
  html +=
    '</tbody></table>' +
    '<div class="encart"><strong>Barème.</strong> Réussite du premier coup sans indice : +20 % d\'XP. Chaque indice : −10 % (jusqu\'à −60 %). Correction consultée avant de réussir : XP divisés par 4. ' +
    'Trois étoiles : sans indice et en 3 essais au plus. Deux étoiles : 2 indices au plus et 6 essais au plus.</div>' +
    '<div class="sans-impression"><button class="bouton primaire" data-action="imprimer">🖨 Imprimer ou enregistrer le bilan (PDF)</button>' +
    '<button class="bouton danger" data-action="reinit-tout">Remettre le score à zéro</button></div>';
  return html;
}

function rendre() {
  rendreBarres();
  rendreLateral();
  const enQuete = E.vue === 'quete';
  $('#vue-quete').hidden = !enQuete;
  $('#vue-page').hidden = enQuete;
  if (enQuete) {
    rendreConsignes();
    rendreAtelier();
  } else {
    const page = $('#vue-page');
    page.innerHTML = E.vue === 'cours' ? pageCours() : E.vue === 'memo' ? pageMemo() : pageScore();
    page.scrollTop = 0;
  }
}

function chargerEditeur() {
  sauver();
  editeur.valeur = E.av[E.id][E.fichier === 'exemple' ? 'exemple' : 'code'];
}

// ---------- navigation ----------

function ouvrirQuete(id, onglet) {
  sauver();
  if (id !== E.id) {
    E.sortie = null;
    E.resultats = null;
    E.reponse = null;
    E.panneau = 'sortie';
    const premiere = E.quetes.find((q) => q.id === id).requetes[0];
    E.req = premiere ? { ...premiere } : { methode: 'GET', chemin: '/', corps: '', entetes: '' };
  }
  E.id = id;
  E.vue = 'quete';
  E.onglet = onglet || (E.av[id].fait || E.av[id].essais > 0 ? 'defi' : 'cours');
  E.fichier = E.onglet === 'cours' ? 'exemple' : 'defi';
  if (E.panneau !== 'tests') E.panneau = enServeur() ? 'requete' : 'sortie';
  memoriser();
  rendre();
  chargerEditeur();
}

function changerVue(vue) {
  sauver();
  // Recliquer sur la section active replie ou déplie la liste (écrans étroits).
  if (vue === E.vue) document.body.classList.toggle('lateral-ouvert');
  else document.body.classList.remove('lateral-ouvert');
  // Un cours se lit librement, mais l'éditeur ne s'ouvre que sur une quête débloquée.
  if (vue === 'quete' && !debloquee(rang(E.id))) return ouvrirQuete(E.quetes[premiereAFaire()].id);
  E.vue = vue;
  memoriser();
  rendre();
  if (vue === 'quete') chargerEditeur();
}

function changerFichier(fichier) {
  if (fichier === E.fichier) return;
  E.fichier = fichier;
  if (quete().serveur) E.panneau = fichier === 'defi' ? 'requete' : 'sortie';
  rendreAtelier();
  chargerEditeur();
}

let minuterieToast = null;
function toast(html) {
  const t = $('#toast');
  t.innerHTML = html;
  t.hidden = false;
  clearTimeout(minuterieToast);
  minuterieToast = setTimeout(() => (t.hidden = true), 5000);
}

// ---------- actions ----------

async function avecServeur(action) {
  if (E.occupe) return;
  clearTimeout(minuterie);
  aSauver = null;
  E.occupe = true;
  rendreAtelier();
  try {
    await action();
  } catch (e) {
    E.sortie = 'Le serveur apilab ne répond pas : ' + e.message + '\nVérifie que « apilab quest » tourne toujours dans le terminal.';
    E.panneau = 'sortie';
  }
  E.occupe = false;
  rendre();
}

function appliquer(reponse) {
  if (reponse.avancement) E.av[E.id] = reponse.avancement;
  if (reponse.score) E.score = reponse.score;
}

const executer = () =>
  avecServeur(async () => {
    const r = await api('executer', { id: E.id, fichier: E.fichier, code: editeur.valeur });
    E.sortie = r.sortie + (r.expire ? '\n⏱ Arrêté après 5 secondes : boucle infinie ?' : r.codeSortie ? `\n(le programme s'est terminé avec le code ${r.codeSortie})` : '');
    E.panneau = 'sortie';
  });

const envoyerRequete = () =>
  avecServeur(async () => {
    E.panneau = 'requete';
    const avecJeton = (texte) => texte.replaceAll('{{token}}', E.jeton || '(aucun token : fais d\'abord POST /auth/login)');
    const r = E.req;
    E.reponse = await api('requete', {
      id: E.id,
      code: E.av[E.id].code,
      methode: r.methode,
      chemin: avecJeton(r.chemin),
      corps: avecJeton(r.corps),
      entetes: avecJeton(r.entetes),
    });
    try {
      const jeton = JSON.parse(E.reponse.reponse.corps).token;
      if (typeof jeton === 'string') E.jeton = jeton;
    } catch {}
  });

const verifier = () =>
  avecServeur(async () => {
    const r = await api('verifier', { id: E.id, code: editeur.valeur });
    appliquer(r);
    E.resultats = r.resultats;
    E.sortie = r.sortie;
    E.panneau = 'tests';
    if (r.xpGagne > 0) {
      E.onglet = 'defi';
      toast(`🎉 <strong>Quête validée !</strong> +${r.xpGagne} XP ${etoiles(r.avancement.etoiles)}<br>Rang : ${esc(r.score.rang)}`);
      // Dernière quête validée : on affiche le bilan final.
      if (r.score.faites === r.score.total) {
        E.vue = 'score';
        memoriser();
      }
    }
  });

const reinitialiser = () => {
  const nom = E.fichier === 'exemple' ? 'exemple.js' : 'defi.js';
  if (!confirm(`Remettre ${nom} dans son état de départ ? Ton code actuel sera perdu.`)) return;
  avecServeur(async () => {
    appliquer(await api('reinit', { id: E.id, fichier: E.fichier }));
    editeur.valeur = E.av[E.id][E.fichier === 'exemple' ? 'exemple' : 'code'];
  });
};

const ACTIONS = {
  exemple: () => changerFichier('exemple'),
  defi: () => ouvrirQuete(E.id, 'defi'),
  ouvrir: () => ouvrirQuete(E.id, 'cours'),
  suivante: () => ouvrirQuete(E.quetes[rang(E.id) + 1].id),
  score: () => changerVue('score'),
  profil: ouvrirInscription,
  imprimer: () => window.print(),
  indice: () => avecServeur(async () => appliquer(await api('indice', { id: E.id }))),
  solution: () => {
    if (!confirm('Voir la correction divise par 4 les XP de cette quête. Continuer ?')) return;
    avecServeur(async () => appliquer(await api('solution', { id: E.id })));
  },
  'reinit-tout': () => {
    if (!confirm('Remettre tout le score à zéro ? Ton code est conservé, mais XP, étoiles et quêtes validées sont effacés.')) return;
    avecServeur(async () => {
      const etat = await api('reinit-tout', {});
      E.av = etat.avancement;
      E.score = etat.score;
    });
  },
};

// ---------- branchements ----------

function brancher() {
  const surClic = (selecteur, attribut, fn) =>
    $(selecteur).addEventListener('click', (e) => {
      const b = e.target.closest(`button[data-${attribut}]`);
      if (b && !b.disabled) fn(b.dataset[attribut]);
    });

  surClic('#barre-activite', 'vue', changerVue);
  surClic('#lateral', 'id', (id) => {
    document.body.classList.remove('lateral-ouvert');
    if (E.vue === 'cours') {
      E.id = id;
      memoriser();
      rendre();
    } else {
      ouvrirQuete(id);
    }
  });
  surClic('#onglets-consignes', 'cle', (cle) => {
    E.onglet = cle;
    rendreConsignes();
    if (cle === 'defi') changerFichier('defi');
  });
  surClic('#onglets-fichiers', 'cle', changerFichier);
  surClic('#onglets-panneau', 'cle', (cle) => {
    E.panneau = cle;
    rendrePanneau();
  });
  surClic('#contenu-consignes', 'action', (a) => ACTIONS[a]());
  surClic('#vue-page', 'action', (a) => ACTIONS[a]());

  $('#btn-executer').addEventListener('click', () => (enServeur() ? envoyerRequete() : executer()));

  // Onglet Requête : le formulaire vit dans E.req pour survivre aux réaffichages.
  surClic('#contenu-panneau', 'req', (i) => {
    E.req = { ...quete().requetes[i] };
    rendrePanneau();
  });
  surClic('#contenu-panneau', 'envoi', envoyerRequete);
  $('#contenu-panneau').addEventListener('input', (e) => {
    if (e.target.dataset.champ) E.req[e.target.dataset.champ] = e.target.value;
  });
  $('#contenu-panneau').addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && e.target.dataset.champ === 'chemin') envoyerRequete();
  });
  $('#btn-verifier').addEventListener('click', verifier);
  $('#btn-reinit').addEventListener('click', reinitialiser);
  $('#btn-theme').addEventListener('click', basculerTheme);
  $('#form-inscription').addEventListener('submit', validerInscription);
  $('#annuler-inscription').addEventListener('click', () => ($('#inscription').hidden = true));
  // Le bilan s'imprime toujours en clair, quel que soit le thème affiché.
  let themeAvantImpression = null;
  window.addEventListener('beforeprint', () => {
    themeAvantImpression = document.documentElement.dataset.theme;
    document.documentElement.dataset.theme = 'light';
  });
  window.addEventListener('afterprint', () => {
    if (themeAvantImpression) document.documentElement.dataset.theme = themeAvantImpression;
  });
  $('#score-haut').addEventListener('click', () => {
    if (E.vue !== 'score') changerVue('score');
  });

  document.addEventListener('keydown', (e) => {
    if (!(e.ctrlKey || e.metaKey) || E.vue !== 'quete') return;
    if (e.key === 'Enter') {
      e.preventDefault();
      if (enServeur()) envoyerRequete();
      else executer();
    } else if (e.key.toLowerCase() === 's') {
      e.preventDefault();
      if (E.fichier === 'defi') verifier();
      else executer();
    }
  });
  window.addEventListener('beforeunload', sauver);
}

async function demarrer() {
  editeur = new Editeur($('#editeur'), surSaisie);
  brancher();
  rendreTheme();
  let etat;
  try {
    etat = await api('etat');
  } catch (e) {
    $('#vue-quete').hidden = true;
    $('#vue-page').hidden = false;
    $('#vue-page').innerHTML = `<h1>Serveur injoignable</h1><p>Lance « apilab quest » dans un terminal, puis recharge cette page.</p><p class="muet">${esc(e.message)}</p>`;
    return;
  }
  Object.assign(E, { parcours: etat.parcours, profil: etat.profil, quetes: etat.quetes, av: etat.avancement, score: etat.score, node: etat.node, dossier: etat.dossier });
  const s = souvenir();
  // Reprendre là où on s'était arrêté, sinon à la première quête non terminée.
  const reprise = E.quetes.findIndex((q) => q.id === s.id);
  const i = reprise >= 0 && debloquee(reprise) ? reprise : premiereAFaire();
  E.id = E.quetes[i].id;
  if (['cours', 'memo', 'score'].includes(s.vue)) {
    E.vue = s.vue;
    rendre();
  } else {
    ouvrirQuete(E.id);
  }
  if (!E.profil) ouvrirInscription();
}

demarrer();
