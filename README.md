# Technical Architecture: A Spanish-Focused Statistical Retrieval Engine

This program implements a classic Vector Space Model (VSM) for information retrieval, specifically tailored for the Spanish language. It is a non-learning, algorithmic system that statistically ranks sentence fragments from a source PDF against a user query. The architecture is built on a different stack than previous versions, notably incorporating stemming for more robust term matching.

The system's operation is divided into an offline indexing phase and an online querying phase.

## 1. Offline Indexing Phase (Corpus Preparation)

This initial phase processes the source PDF into a searchable, numerical format.

* Text Extraction: The github.com/ledongthuc/pdf library is used to open and read the PDF file. It extracts the raw text content from the entire document into a single string.

* Text Segmentation (Sentence Splitting):
    * Why: To provide granular answers, the full text is segmented into smaller units. Instead of paragraphs, this version targets sentences.
    * How: A custom function, splitIntoSentences, performs this segmentation. It uses a simple but effective heuristic: it splits the entire text block by the period character (.). To avoid incorrectly splitting on abbreviations, it maintains a hardcoded map of common Spanish abbreviations (e.g., art., sr.). If a split occurs after one of these known abbreviations, it attempts to merge the fragment back with the previous sentence.

* Text Preprocessing & Vectorization: Each sentence (now a "document" in our corpus) is converted into a numerical vector through a multi-step pipeline.
    1.  Normalization: The sentence is converted to lowercase. Newline characters are replaced with spaces to ensure proper tokenization.
    2.  Tokenization: The normalized string is split into tokens (words). The function strings.FieldsFunc is used with a custom rule to define a token as a sequence of alphabetic characters, including Spanish-specific accented vowels and the letter ñ.
    3.  Stop Word Removal: A hardcoded map[string]bool of common Spanish stop words is used to filter out high-frequency, low-meaning words from the token list.
    4.  Stemming: This is a key step in this version. The github.com/kljensen/snowball library is invoked with the "spanish" algorithm. Stemming reduces words to their root or base form (e.g., contratos, contratista, and contratación might all be reduced to the stem contrat). This allows the system to match related words, making it more robust than simple keyword matching. The final list of tokens for a document is a list of its stems.
    5.  TF-IDF Calculation: A custom tfidf function builds the final vectors.
        * A vocabulary map[string]int is created, mapping every unique stem in the corpus to an integer index.
        * Term Frequency (TF): For each sentence, a vector is created. The raw count of each stem is calculated.
        * Document Frequency (DF): The system counts how many sentences contain each stem.
        * TF-IDF Weighting: The raw TF scores are weighted by the Inverse Document Frequency. The formula used is TF * log(N / (1 + DF)), where N is the total number of sentences. This weights stems that appear frequently in one sentence but rarely across the whole document.
        * Vector Normalization: Each final document vector is normalized using the L2 norm (Euclidean length). This ensures that the length of the sentence does not affect its similarity score.

## 2. Online Querying Phase

This phase executes once per run to find the best answer for a single user query.

1.  Query Processing: The user's query string is processed through the *exact same pipeline* as the documents: lowercasing, stop word removal, and, critically, Spanish stemming.
2.  Query Vectorization: A TF-IDF vector is constructed for the stemmed query using the same vocabulary and DF statistics gathered during the indexing phase. This vector is also normalized.
3.  Similarity Calculation: Cosine Similarity is used to calculate the similarity between the normalized query vector and every normalized sentence vector.
4.  Ranking and Retrieval: The sentences are ranked by their cosine similarity score. The single top-scoring result is displayed, but only if its score exceeds a hardcoded threshold of 0.2, filtering out poor matches.

---

## Language Specificity: Spanish Only

This program is exclusively designed to work with Spanish documents and will perform poorly or fail entirely with other languages, such as English. This is not a configurable option; it is fundamental to its architecture for the following reasons:

1.  Hardcoded Stop Words: The stopwords map contains only Spanish words. When processing an English document, it would fail to remove English stop words, skewing the TF-IDF weights.
2.  Mandatory Spanish Stemming: The line snowball.Stem(word, "spanish", true) explicitly invokes the Snowball stemming algorithm that is mathematically designed for the morphology and grammar of the Spanish language. Applying this to English words would result in nonsensical, incorrectly-chopped "stems," making matching impossible.
3.  Spanish-Specific Tokenization & Segmentation: The regex for tokenization includes Spanish characters (á, é, ñ, etc.). The sentence-splitting logic relies on a list of Spanish abbreviations.

## Inherent Shortcomings and Fundamental Limitations

Despite the addition of stemming, the system shares the core limitations of any non-AI, keyword-based retrieval model.

1.  Absence of True Semantic Understanding: The system still operates on a "bag-of-words" (or "bag-of-stems") principle. Stemming provides a rudimentary form of lexical relation (connecting contrato and contratar), but it has zero understanding of actual meaning. It cannot grasp synonyms (e.g., ley vs. normativa), context, or intent.

2.  Extractive Nature: The program can only display verbatim sentences that exist in the source PDF. It cannot synthesize an answer from multiple sentences, provide a summary, or answer any question that requires inferential reasoning. The answer is always a direct, and sometimes out-of-context, quote.

3.  "Closed-World" Constraint: The model's knowledge is limited to the single PDF provided. It has no external or common-sense knowledge.

4.  Answer Fragmentation: Since segmentation occurs at the sentence level, an answer that spans two or more sentences in the original text will be fragmented. The system might return the first sentence as a high-scoring match while the crucial second sentence is missed entirely.

5.  Sensitivity to Wording (Partially Mitigated): Stemming makes the system more robust than a pure keyword matcher. A search for contratar can now match documents containing contratos. However, it is still sensitive to terminology that does not share a common stem. A query for *"reglamento"* will not match a sentence that uses the synonym *"normativa"*. The user must still have some idea of the vocabulary used in the document.
