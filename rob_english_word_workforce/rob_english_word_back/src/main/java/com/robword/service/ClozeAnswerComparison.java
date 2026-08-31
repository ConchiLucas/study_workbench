package com.robword.service;

import java.text.Normalizer;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

/**
 * A single indexed comparison shared by sentence-level and word-level review updates.
 */
public record ClozeAnswerComparison(
        List<String> answers,
        List<String> expectedWords,
        List<Integer> wrongIndexes,
        boolean correct
) {

    public ClozeAnswerComparison {
        answers = List.copyOf(answers);
        expectedWords = List.copyOf(expectedWords);
        wrongIndexes = List.copyOf(wrongIndexes);
    }

    public static ClozeAnswerComparison compare(List<String> rawAnswers, List<String> rawExpectedWords) {
        List<String> answers = sanitize(rawAnswers);
        List<String> expectedWords = sanitize(rawExpectedWords);
        boolean mismatchedSize = answers.size() != expectedWords.size();
        List<Integer> wrongIndexes = new ArrayList<>();

        for (int index = 0; index < expectedWords.size(); index++) {
            String answer = index < answers.size() ? answers.get(index) : "";
            if (mismatchedSize || !normalize(answer).equals(normalize(expectedWords.get(index)))) {
                wrongIndexes.add(index);
            }
        }

        return new ClozeAnswerComparison(
                answers,
                expectedWords,
                wrongIndexes,
                !mismatchedSize && wrongIndexes.isEmpty()
        );
    }

    /**
     * Revealing an answer is an explicit learning miss, even if the client sends
     * the revealed words back in the payload. Keep the submitted values for the
     * answer record while marking every expected blank as wrong.
     */
    public static ClozeAnswerComparison reveal(List<String> rawAnswers, List<String> rawExpectedWords) {
        List<String> answers = sanitize(rawAnswers);
        List<String> expectedWords = sanitize(rawExpectedWords);
        List<Integer> wrongIndexes = new ArrayList<>(expectedWords.size());
        for (int index = 0; index < expectedWords.size(); index++) {
            wrongIndexes.add(index);
        }
        return new ClozeAnswerComparison(answers, expectedWords, wrongIndexes, false);
    }

    private static List<String> sanitize(List<String> values) {
        if (values == null) {
            return List.of();
        }
        return values.stream()
                .map(value -> value == null ? "" : value.trim())
                .toList();
    }

    private static String normalize(String value) {
        return Normalizer.normalize(value, Normalizer.Form.NFKC).toLowerCase(Locale.ROOT);
    }
}
