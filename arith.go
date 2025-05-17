package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Constantes pour la précision et les limites du codage arithmétique
const (
	Precision       = 40
	TopValue        = (1 << Precision) - 1
	EOF             = 256 // Symbole de fin de fichier
	TotalSymbolFreq = 257 // 256 symboles + EOF
)

// Interface pour la compression
type Compressor interface {
	Compress([]byte) []byte
}

// Interface pour la décompression
type Decompressor interface {
	Decompress([]byte) ([]byte, error)
}

// Modèle adaptatif avec fréquences symboliques
type Model struct {
	Freq     [TotalSymbolFreq + 1]uint64 // Fréquences individuelles
	CumFreqs [TotalSymbolFreq + 2]uint64 // Fréquences cumulées
}

// Initialise un modèle avec des fréquences uniformes
func NewInitialModel() *Model {
	m := &Model{}
	for i := 0; i <= EOF; i++ {
		m.Freq[i] = 1
	}
	m.updateCumulative()
	return m
}

// Met à jour les fréquences cumulées à partir des fréquences individuelles
func (m *Model) updateCumulative() {
	m.CumFreqs[0] = 0
	for i := 1; i <= EOF+1; i++ {
		m.CumFreqs[i] = m.CumFreqs[i-1] + m.Freq[i-1]
	}
}

// Implémentation principale du codeur arithmétique
type ArithmeticCoder struct{}


// Compress compresse les données en utilisant le codage arithmétique adaptatif.
// Retourne un tableau d'octets compressés.
func (a *ArithmeticCoder) Compress(data []byte) []byte {
	intData := make([]int, len(data)+1)
	for i, b := range data {
		intData[i] = int(b)
	}
	intData[len(data)] = EOF

	// Initialisation du modèle de fréquences
	model := NewInitialModel()

	// Limites initiales de l'intervalle [low, high]
	low := uint64(0x0000000000)
	high := uint64(TopValue)

	output := make([]byte, 0)   
	removedBytes := 0           
	var savedLow uint64        

	logFile, err := os.Create("results/" + "log_encode.txt")
	if err != nil {
		panic(fmt.Sprintf("Erreur lors de la création du fichier de log : %v", err))
	}
	defer logFile.Close()

	for _, sym := range intData {
		total := model.CumFreqs[EOF+1]     // fréquence cumulée totale
		lowFreq := model.CumFreqs[sym]     // fréquence basse pour le symbole courant
		highFreq := model.CumFreqs[sym+1]  // fréquence haute pour le symbole courant

		rangeSize := high - low + 1
		// Mise à jour des bornes selon la fréquence
		newLow := low + (rangeSize*lowFreq)/total
		newHigh := low + (rangeSize*highFreq)/total - 1

		// Écriture des informations dans le fichier de log
		fmt.Fprintf(logFile, "\n\n>> octet 0x%02x=%d (%c):\n", sym, sym, printableRune(sym))
		fmt.Fprintf(logFile, "   low  = 0x%010x\n", low)
		fmt.Fprintf(logFile, "   high = 0x%010x\n", high)
		fmt.Fprintf(logFile, "   intervalle de l'octet dans le modèle = [%d, %d[ pour un total cumulé de %d\n", lowFreq, highFreq, total)

		// Actualisation des bornes
		low = newLow
		high = newHigh

		fmt.Fprintf(logFile, ">> mise à jour de low et high:\n")
		fmt.Fprintf(logFile, "   low  = 0x%010x\n", low)
		fmt.Fprintf(logFile, "   high = 0x%010x\n", high)

		// Compression par décalages successifs des bornes si possible
		for {
			msbLow := low >> 32
			msbHigh := high >> 32

			if msbLow == msbHigh {
				// Les octets de poids fort sont égaux, on peut émettre ce byte
				fmt.Fprintf(logFile, ">> Les octets de poids fort sont égaux !\n")
				fmt.Fprintf(logFile, "   On ajoute 0x%02x dans le résultat.\n", msbLow)

				output = append(output, byte(msbLow))
				low = (low << 8) & TopValue
				high = ((high << 8) | 0xFF) & TopValue

				fmt.Fprintf(logFile, "   Après décalage:\n")
				fmt.Fprintf(logFile, "     low  = 0x%010x\n", low)
				fmt.Fprintf(logFile, "     high = 0x%010x\n", high)
			} else if (msbLow+1 == msbHigh) &&
				((low>>24)&0xFF) == 0xFF &&
				((high>>24)&0xFF) == 0x00 {
				// Cas d'underflow potentiel : les octets de poids fort sont presque égaux
				fmt.Fprintf(logFile, ">> Les octets de poids fort sont presque égaux, et les octets suivants sont 0xff / 0x00 !\n")
				fmt.Fprintf(logFile, ">> On garde les octets de poids fort, mais on supprime les suivants.\n")

				output = append(output, byte(msbLow))

				b0 := (low >> 32) & 0xFF
				b2 := (low >> 16) & 0xFF
				b3 := (low >> 8) & 0xFF
				b4 := low & 0xFF
				low = (b0 << 32) | (b2 << 24) | (b3 << 16) | (b4 << 8) | 0x00

				b0 = (high >> 32) & 0xFF
				b2 = (high >> 16) & 0xFF
				b3 = (high >> 8) & 0xFF
				b4 = high & 0xFF
				high = (b0 << 32) | (b2 << 24) | (b3 << 16) | (b4 << 8) | 0xFF

				fmt.Fprintf(logFile, ">> mise à jour de low et high:\n")
				fmt.Fprintf(logFile, "   low  = 0x%010x\n", low)
				fmt.Fprintf(logFile, "   high = 0x%010x\n", high)
			} else {
				break
			}
		}

		// Mise à jour des fréquences dans le modèle pour s'adapter aux données
		model.Freq[sym]++
		model.updateCumulative()
	}

	// Finalisation : ajout des derniers octets basés sur la borne basse 'low'
	fmt.Fprintf(logFile, ">> Finalisation, on ajoute les octets de low dans le résultat :\n")

	for i := 0; i < removedBytes; i++ {
		shift := 8 * (4 - i)
		octet := byte((savedLow >> shift) & 0xFF)
		output = append(output, octet)
		fmt.Fprintf(logFile, "   0x%02x ajouté (de la sauvegarde)\n", octet)
	}
	for i := removedBytes; i < 5; i++ {
		shift := 8 * (4 - i)
		octet := byte((low >> shift) & 0xFF)
		output = append(output, octet)
		fmt.Fprintf(logFile, "   0x%02x ajouté\n", octet)
	}

	fmt.Fprintf(logFile, "Résultat compressé (en hex) : % x\n", output)

	return output
}


// printableRune retourne le caractère imprimable ou un point pour les octets non imprimables.
func printableRune(sym int) rune {
	if sym >= 32 && sym <= 126 {
		return rune(sym)
	}
	return '.'
}

// Decompress décompresse les données encodées par le codage arithmétique.
func (a *ArithmeticCoder) Decompress(encoded []byte) ([]byte, error) {
	model := NewInitialModel()

	// Vérification minimale : il faut au moins 5 octets pour initialiser la valeur du code
	if len(encoded) < 5 {
		return nil, fmt.Errorf("données compressées trop courtes")
	}

	// Initialisation des bornes et du code actuel
	low := uint64(0)
	high := uint64(TopValue)
	code := (uint64(encoded[0]) << 32) | (uint64(encoded[1]) << 24) |
		(uint64(encoded[2]) << 16) | (uint64(encoded[3]) << 8) | uint64(encoded[4])
	pos := 5

	decoded := make([]byte, 0)

	for {
		rangeSize := high - low + 1
		total := model.CumFreqs[EOF+1]

		// Calcul de la valeur dans le modèle cumulatif
		value := ((code - low + 1) * total - 1) / rangeSize

		// Recherche du symbole correspondant à cette valeur
		sym := -1
		for i := 0; i <= EOF; i++ {
			if model.CumFreqs[i] <= value && value < model.CumFreqs[i+1] {
				sym = i
				break
			}
		}
		if sym == -1 {
			return nil, fmt.Errorf("symbole introuvable lors du décodage")
		}

		// EOF = fin du message
		if sym == EOF {
			break
		}

		// Ajout du symbole décodé au résultat
		decoded = append(decoded, byte(sym))

		// Mise à jour des bornes selon le symbole décodé
		newLow := low + (rangeSize*model.CumFreqs[sym])/total
		newHigh := low + (rangeSize*model.CumFreqs[sym+1])/total - 1
		low = newLow
		high = newHigh

		// Mise à jour du modèle avec le nouveau symbole
		model.Freq[sym]++
		model.updateCumulative()

		// Décalage des bits si les octets de poids fort de low et high sont égaux
		for {
			msbLow := low >> 32
			msbHigh := high >> 32

			if msbLow == msbHigh {
				if pos >= len(encoded) {
					return nil, fmt.Errorf("fin inattendue des données pendant le décodage")
				}
				nextByte := uint64(encoded[pos])
				pos++

				// Décalage à gauche des bornes et du code, ajout d'un nouvel octet
				low = (low << 8) & TopValue
				high = ((high << 8) | 0xFF) & TopValue
				code = ((code << 8) | nextByte) & TopValue
			} else {
				break
			}
		}
	}

	return decoded, nil
}

func CompressDecompress(data []byte) ([]byte, []byte, error) {
	ac := &ArithmeticCoder{}
	compressed := ac.Compress(data)
	decompressed, err := ac.Decompress(compressed)
	if err != nil {
		return compressed, nil, err
	}
	return compressed, decompressed, nil
}

func TestCompression(inputPath string) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		log.Fatalf("Erreur de lecture du fichier : %v", err)
	}

	fmt.Println("Début du test de compression/décompression...")

	ac := &ArithmeticCoder{}

	// Mesure du temps de compression
	compressionStart := time.Now()
	compressed := ac.Compress(data)
	compressionDuration := time.Since(compressionStart)

	// Sauvegarde du fichier compressé
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	compressedPath := "results/" + baseName + ".out"
	err = os.WriteFile(compressedPath, compressed, 0644)
	if err != nil {
		fmt.Printf("Erreur d'écriture du fichier compressé : %v\n", err)
	}

	originalSize := len(data)
	compressedSize := len(compressed)
	gain := 100.0 * (1.0 - float64(compressedSize)/float64(originalSize))

	fmt.Printf("Test terminé.\n")
	fmt.Printf("Fichier original :      %s (%d octets)\n", inputPath, originalSize)
	fmt.Printf("Fichier compressé :     %s (%d octets)\n", compressedPath, compressedSize)
	fmt.Printf("Gain d'espace :         %.2f %%\n", gain)
	fmt.Printf("Temps de compression :  %s\n", compressionDuration)

	// Mesure du temps de décompression
	decompressionStart := time.Now()
	decompressed, err := ac.Decompress(compressed)
	decompressionDuration := time.Since(decompressionStart)

	if err != nil {
		fmt.Printf("Erreur lors de la décompression : %v\n", err)
		fmt.Println("Comparaison impossible, arrêt de la vérification.")
	} else {
		// Vérification de la validité des données décompressées
		if bytes.Equal(data, decompressed) {
			decodedPath := "results/" + baseName + "_decoded.txt"
			err = os.WriteFile(decodedPath, decompressed, 0644)
			if err != nil {
				log.Printf("Erreur d'écriture du fichier décompressé : %v", err)
			} else {
				fmt.Printf("Fichier décompressé :  %s\n", decodedPath)
				fmt.Printf("Temps de décompression : %s\n", decompressionDuration)
			}
		} else {
			fmt.Println("Les données décompressées ne correspondent pas aux données originales.")
			
		}
	}

}


func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <input_file>")
	}
	inputPath := os.Args[1]
	TestCompression(inputPath)
}
