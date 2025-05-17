# Projet de Codage Arithmétique

## Objectif

Ce projet implémente un **algorithme de codage arithmétique** en Go. Il permet de **compresser** et **décompresser** des fichiers binaires, tout en conservant une grande précision dans le calcul des intervalles. Le programme inclut des fichiers de log pour suivre les étapes de compression, et fournit un test automatique avec mesures de performance.

---


### Version Initiale

La première version du projet utilisait une implémentation naïve avec :

* Un encodage arithmétique sans table de fréquence dynamique.
* Un alphabet limité, ce qui empêchait de traiter des octets binaires complets.
* Une précision insuffisante pour de longues chaînes de données.

### Version Finale

La version actuelle repose sur :

* Un alphabet de 257 symboles : les octets de 0 à 255, plus un symbole spécial `EOF` (fin de fichier).
* Une précision de 40 bits, avec une constante `TopValue = 0xFFFFFFFFFF`.
* Une mise à jour dynamique des fréquences après chaque symbole.
* Des logs très détaillés pour le suivi étape par étape.
* Un système de test automatique qui mesure :
  * Le temps de compression/décompression
  * Le taux de compression (gain en espace)

---

## Structure Générale

### Interfaces

Le projet implémente deux interfaces :

```go
type Compresser interface {
    Compress([]byte) []byte
}

type Decompresser interface {
    Decompress([]byte) []byte
}
```

Un seul objet peut satisfaire aux deux interfaces, car les noms sont différents mais les signatures sont symétriques.

---

## Modèle de Fréquences

```go
type Model struct {
    Freq     [257 + 1]uint64     // Fréquences individuelles
    CumFreqs [257 + 2]uint64     // Fréquences cumulées
}
```

* Chaque symbole commence avec une fréquence initiale de 1 (modèle uniforme).
* Après chaque symbole traité, la fréquence du symbole est augmentée.
* Le tableau cumulé (`CumFreqs`) est mis à jour pour refléter les nouvelles probabilités.

---

## Encodage Arithmétique

### Étapes :

1. On initialise `low = 0x0000000000` et `high = 0xFFFFFFFFFF`.
2. Pour chaque symbole :
   * On réduit l’intervalle `[low, high]` en fonction des fréquences.
   * On détermine si les octets de poids fort de `low` et `high` sont identiques :
     * Si oui → on émet l’octet commun.
     * Si « presque » égaux → on applique une optimisation spéciale (cas `0xFF / 0x00`).
3. Après tous les symboles, on ajoute les octets restants pour compléter l’encodage.

### Exemple de log :

```
>> octet 0x62 = 98 (b):
   low  = 0x0000000000
   high = 0xffffffffff
   mise à jour:
   low  = 0x000002aaaa
   high = 0x0000055555
```

---

## Décodage

### Initialisation :

* `code` est construit à partir des 5 premiers octets du fichier compressé.
* Ensuite, à chaque itération :
  * On trouve le symbole correspondant à la valeur `code` dans l’intervalle `[low, high]`.
  * On met à jour `low`, `high` et `code`.
  * On décale si possible pour lire un nouvel octet.

### Fin du décodage :

* Le décodeur s’arrête lorsque le symbole `EOF` est rencontré.

---

## Test Automatique

Le fichier `arith.go` contient une exécution automatique qui :

1. Lit le fichier.
2. Encode le contenu + EOF.
3. Décode le résultat compressé.
4. Vérifie que les données originales et décompressées sont **identiques**.
5. Affiche les statistiques.

---

## Cas Particuliers Gérés

* Fin prématurée du fichier compressé → message d’erreur.
* Intervalle trop petit → re-normalisation par décalage.
* Octets presque égaux (ex. `0xFFFF...` et `0x0000...`) → gestion spécifique.
* Surcroît de fréquence (overflow) → évité car les entiers sont suffisamment larges (`uint64`).

---

## Instructions d'Utilisation

### Compilation

Lancez simplement le programme avec :

```bash
go run main.go chemin/vers/fichier.txt
```

#### Par example: 
```bash
go run main.go example/banane
```

## Problématique

Un algorithme de compression et de décompression a été implémenté et fonctionne sur de petits fichiers simples (par exemple `banane`).  
Cependant, le fichier compressé est souvent **plus volumineux** que l’original, ce qui donne un **taux de compression négatif**. Cela s’explique par :
- L’ajout d’un en-tête initial de 5 octets,
- Une table de fréquences initiale uniforme,
- Le manque d’efficacité sur de très courtes séquences.

**Toutefois, sur de grands fichiers, l’algorithme montre une amélioration significative, avec un taux de compression pouvant atteindre environ 50 à 70 %.**  

Fichier original :      test_files/calvin-P3.ppm (2045075 octets)
Fichier compressé :     results/calvin-P3.out (609554 octets)
Gain d'espace :         70.19 %

Fichier original :      test_files/karamazov.txt (2042003 octets)
Fichier compressé :     results/karamazov.out (1184859 octets)
Gain d'espace :         41.98 %

En comparant les logs de compression d’un fichier comme `aaa.txt` avec ceux d’un exemple de référence, on peut supposer que la **compression fonctionne également sur des cas plus complexes**.  
Néanmoins, des **erreurs apparaissent lors de la décompression**, souvent dès le **premier symbole**, à cause d’un **décalage binaire incorrect?**.

## Remarque sur la mesure du temps

Le programme affiche la durée d’exécution pour la compression et la décompression.  
**Ce temps inclut aussi les opérations d’écriture dans le fichier**, il ne reflète donc pas uniquement l’efficacité de l’algorithme.
