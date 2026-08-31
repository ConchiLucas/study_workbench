from word_agent.services.best_sentence_cloze import build_best_sentence_cloze


def test_uses_inflected_third_person_form_as_answer() -> None:
    result = build_best_sentence_cloze(
        "value",
        "The company values fairness in its hiring process.",
    )

    assert result is not None
    assert result.answer == "values"
    assert result.cloze_sentence == "The company ____ fairness in its hiring process."
    assert result.match_kind == "inflected"


def test_preserves_sentence_capitalization() -> None:
    result = build_best_sentence_cloze("april", "April is usually warm here.")

    assert result is not None
    assert result.answer == "April"
    assert result.cloze_sentence == "____ is usually warm here."


def test_handles_past_plural_continuous_and_irregular_forms() -> None:
    cases = [
        ("abash", "The criticism abashed him.", "abashed"),
        ("allergy", "Seasonal allergies are common.", "allergies"),
        ("abet", "He was accused of abetting them.", "abetting"),
        ("write", "She wrote a short note.", "wrote"),
        ("caveman", "Tools were used by cavemen.", "cavemen"),
        ("outgrow", "He has outgrown his coat.", "outgrown"),
        ("overhear", "I overheard their conversation.", "overheard"),
        ("oversleep", "She overslept this morning.", "overslept"),
        ("devilish", "The puzzle was devilishly difficult.", "devilishly"),
    ]

    for word, sentence, answer in cases:
        result = build_best_sentence_cloze(word, sentence)
        assert result is not None
        assert result.answer == answer
        assert result.cloze_sentence.count("____") == 1


def test_uses_primary_anchor_for_pattern_word() -> None:
    result = build_best_sentence_cloze(
        "call(sb)back",
        "Could you ask him to call me back later?",
    )

    assert result is not None
    assert result.answer == "call"
    assert result.cloze_sentence == "Could you ask him to ____ me back later?"
    assert result.match_kind == "phrase_anchor"


def test_blanks_every_repeated_occurrence_of_the_same_answer() -> None:
    result = build_best_sentence_cloze("do", "What do you want to do today?")

    assert result is not None
    assert result.answer == "do"
    assert result.cloze_sentence == "What ____ you want to ____ today?"
    assert result.occurrences == 2


def test_rejects_different_candidate_forms_in_one_sentence() -> None:
    assert build_best_sentence_cloze("lie", "He lies often but is lying now.") is None


def test_matches_word_with_trailing_punctuation() -> None:
    result = build_best_sentence_cloze("a.", "The abbreviation a. appears here.")

    assert result is not None
    assert result.answer == "a."
    assert result.cloze_sentence == "The abbreviation ____ appears here."
