'use strict';
// runner.js — lance les tests d'une quête sur le code de l'élève.
// Appelé par « apilab quest » : node runner.js <fichier.quete> <code.js> <resultat.json>

const fs = require('fs');
const path = require('path');
const util = require('util');

const [, , fichierQuete, fichierCode, fichierResultat] = process.argv;

class Echec extends Error {}

const voir = (v) => util.inspect(v, { depth: 5, breakLength: 100, maxArrayLength: 20 });

function section(src, nom) {
  const lignes = [];
  let dedans = false;
  for (const l of src.replace(/\r\n/g, '\n').split('\n')) {
    if (l.startsWith('=== ')) dedans = l.slice(4).trim() === nom;
    else if (dedans) lignes.push(l);
  }
  return lignes.join('\n');
}

// Retire les commentaires et le texte des chaînes : les règles (« pas de boucle for »)
// ne doivent pas se déclencher sur un mot écrit dans un commentaire.
function nettoyer(src) {
  let out = '';
  let i = 0;
  const n = src.length;
  while (i < n) {
    const ch = src[i];
    const suivant = src[i + 1];
    if (ch === '/' && suivant === '/') {
      while (i < n && src[i] !== '\n') i++;
    } else if (ch === '/' && suivant === '*') {
      i += 2;
      while (i < n && !(src[i] === '*' && src[i + 1] === '/')) i++;
      i += 2;
    } else if (ch === '"' || ch === "'") {
      i++;
      while (i < n && src[i] !== ch && src[i] !== '\n') i += src[i] === '\\' ? 2 : 1;
      i++;
      out += ch + ch;
    } else if (ch === '`') {
      i++;
      out += '``';
      while (i < n && src[i] !== '`') {
        if (src[i] === '\\') {
          i += 2;
        } else if (src[i] === '$' && src[i + 1] === '{') {
          // le code dans ${ … } reste du code
          let profondeur = 1;
          i += 2;
          out += ' ';
          while (i < n && profondeur > 0) {
            if (src[i] === '{') profondeur++;
            else if (src[i] === '}') profondeur--;
            if (profondeur > 0) out += src[i];
            i++;
          }
          out += ' ';
        } else {
          i++;
        }
      }
      i++;
    } else {
      out += ch;
      i++;
    }
  }
  return out;
}

const t = {
  egal(obtenu, attendu, quoi) {
    if (!util.isDeepStrictEqual(obtenu, attendu)) {
      throw new Echec(`${quoi ? quoi + '\n' : ''}attendu : ${voir(attendu)}\nobtenu  : ${voir(obtenu)}`);
    }
  },
  proche(obtenu, attendu, quoi) {
    if (typeof obtenu !== 'number' || Math.abs(obtenu - attendu) > 0.005) {
      throw new Echec(`${quoi ? quoi + '\n' : ''}attendu : ${voir(attendu)}\nobtenu  : ${voir(obtenu)}`);
    }
  },
  vrai(condition, message) {
    if (!condition) throw new Echec(message);
  },
  // Appelle fn(donnee) et échoue si la donnée d'origine a été modifiée.
  sansMutation(donnee, fn) {
    const avant = JSON.stringify(donnee);
    const resultat = fn(donnee);
    if (JSON.stringify(donnee) !== avant) {
      throw new Echec(
        `Tu as modifié la donnée d'origine (mutation).\navant : ${avant}\naprès : ${JSON.stringify(donnee)}\n` +
          'Construis une copie au lieu de modifier.'
      );
    }
    return resultat;
  },
};

function ligneEleve(e) {
  const m = String((e && e.stack) || '').match(new RegExp(path.basename(fichierCode).replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ':(\\d+)'));
  return m ? ` (ligne ${m[1]} de ton fichier)` : '';
}

const decrire = (e) => (e instanceof Echec ? e.message : `${(e && e.name) || 'Erreur'} : ${(e && e.message) || e}${ligneEleve(e)}`);

async function principal() {
  const resultats = [];
  const tests = [];
  const regles = [];
  new Function('test', 'interdit', 'requis', section(fs.readFileSync(fichierQuete, 'utf8'), 'tests'))(
    (nom, fn) => tests.push({ nom, fn }),
    (motif, message) => regles.push({ motif, message, interdit: true }),
    (motif, message) => regles.push({ motif, message, interdit: false })
  );

  let mod;
  try {
    mod = require(path.resolve(fichierCode));
  } catch (e) {
    return [{ nom: 'Le fichier se charge sans erreur', ok: false, detail: decrire(e) }];
  }
  if (mod === null || typeof mod !== 'object') mod = {};
  // Un nom oublié dans module.exports donne un message clair plutôt que « undefined is not a function ».
  const m = new Proxy(mod, {
    get(cible, cle) {
      if (typeof cle === 'string' && !(cle in cible)) {
        throw new Echec(`« ${cle} » n'est pas exporté. Vérifie son nom et la ligne module.exports = { … } en bas du fichier.`);
      }
      return cible[cle];
    },
  });

  const propre = nettoyer(fs.readFileSync(fichierCode, 'utf8'));
  for (const r of regles) {
    const trouve = r.motif.test(propre);
    resultats.push({ nom: 'Règle : ' + r.message, ok: r.interdit ? !trouve : trouve });
  }

  for (const { nom, fn } of tests) {
    let minuterie;
    try {
      await Promise.race([
        Promise.resolve().then(() => fn(m, t)),
        new Promise((_, rejeter) => {
          minuterie = setTimeout(() => rejeter(new Echec('Test trop long : promesse jamais résolue ?')), 1500);
        }),
      ]);
      resultats.push({ nom, ok: true });
    } catch (e) {
      resultats.push({ nom, ok: false, detail: decrire(e) });
    } finally {
      clearTimeout(minuterie);
    }
  }
  return resultats;
}

principal()
  .catch((e) => [{ nom: 'Le moteur de tests fonctionne', ok: false, detail: String((e && e.stack) || e) }])
  .then((resultats) => {
    fs.writeFileSync(fichierResultat, JSON.stringify(resultats));
    process.exit(0); // coupe les minuteries que le code de l'élève aurait laissées
  });
