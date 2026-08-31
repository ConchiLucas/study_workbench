package com.robword.mapper;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.condition.EnabledIfEnvironmentVariable;

import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.ArrayList;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;

@EnabledIfEnvironmentVariable(named = "RUN_POSTGRES_INTEGRATION", matches = "true")
class WrongWordQueueEventSqlProviderPostgresTest {

    private static final long USER_ID = 77L;

    @Test
    void executesBestSentenceOnlySemanticsAgainstPostgres() throws SQLException {
        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                execute(connection, "SET LOCAL search_path TO pg_temp");
                createFixtureTables(connection);

                insertProgress(connection, 101L, "alpha");
                insertWord(connection, "alpha", "dictionary alpha sentence");
                insertBestSentence(connection, 10L, "alpha", "best alpha sentence");
                assertRows(connection, "best alpha sentence", "best_sentence", 1L, "progress:101");

                clearFixture(connection);
                insertProgress(connection, 102L, "beta");
                insertWord(connection, "beta", "dictionary beta sentence");
                insertBestSentence(connection, 20L, "beta", "   ");
                assertRows(connection, null, "none", 1L, "progress:102");

                clearFixture(connection);
                insertProgress(connection, 103L, "gamma");
                insertWord(connection, "gamma", "dictionary gamma sentence");
                insertBestSentence(connection, 30L, "gamma", "first best gamma sentence");
                insertBestSentence(connection, 31L, "gamma", "second best gamma sentence");
                assertRows(connection, "first best gamma sentence", "best_sentence", 1L, "progress:103");
            } finally {
                connection.rollback();
            }
        }
    }

    private static Connection openConnection() throws SQLException {
        String url = requiredEnvironment("SPRING_DATASOURCE_URL");
        String username = requiredEnvironment("SPRING_DATASOURCE_USERNAME");
        String password = requiredPassword();
        return DriverManager.getConnection(url, username, password);
    }

    private static String requiredPassword() {
        String value = System.getenv("SPRING_DATASOURCE_PASSWORD");
        if (value == null) {
            throw new IllegalStateException(
                    "SPRING_DATASOURCE_PASSWORD must be set when RUN_POSTGRES_INTEGRATION=true");
        }
        return value;
    }

    private static String requiredEnvironment(String name) {
        String value = System.getenv(name);
        if (value == null || value.isBlank()) {
            throw new IllegalStateException(name + " must be set when RUN_POSTGRES_INTEGRATION=true");
        }
        return value;
    }

    private static void createFixtureTables(Connection connection) throws SQLException {
        execute(connection, """
                CREATE TEMP TABLE game_answer_detail (
                    id bigint, user_id bigint, word_content text, create_time timestamptz,
                    is_correct integer, record_id bigint, word_difficulty integer, answer_time_ms bigint,
                    correct_answer_index integer, option_1 text, option_2 text, option_3 text, option_4 text
                )
                """);
        execute(connection, """
                CREATE TEMP TABLE game_record (
                    id bigint, mode text, training_difficulty_group text, training_difficulty_level text,
                    match_difficulty_group text, match_difficulty_level text, match_difficulty_label text
                )
                """);
        execute(connection, """
                CREATE TEMP TABLE sentence_cloze_answer_record (
                    id bigint, user_id bigint, attempt_no integer, cost_ms bigint, create_time timestamptz,
                    answers_json text, expected_words_json text, cloze_item_id bigint, is_correct boolean
                )
                """);
        execute(connection, """
                CREATE TEMP TABLE sentence_cloze_item (
                    id bigint, source text, provider_label text, source_word_ids_json text
                )
                """);
        execute(connection, """
                CREATE TEMP TABLE word (
                    id bigint, word text, sentence text, status integer, difficulty integer, frequency integer
                )
                """);
        execute(connection, """
                CREATE TEMP TABLE wrong_word_review_progress (
                    id bigint, user_id bigint, word text, normalized_word text, last_wrong_time timestamptz,
                    wrong_count integer, status text, review_stage integer, next_review_time timestamptz
                )
                """);
        execute(connection, """
                CREATE TEMP TABLE word_clean_best_sentence (
                    id bigint, word text, sentence text
                )
                """);
    }

    private static void clearFixture(Connection connection) throws SQLException {
        execute(connection, "DELETE FROM word_clean_best_sentence");
        execute(connection, "DELETE FROM word");
        execute(connection, "DELETE FROM wrong_word_review_progress");
    }

    private static void insertProgress(Connection connection, long id, String word) throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement("""
                INSERT INTO wrong_word_review_progress (
                    id, user_id, word, normalized_word, last_wrong_time, wrong_count, status, review_stage
                ) VALUES (?, ?, ?, ?, now(), 3, 'learning', 1)
                """)) {
            statement.setLong(1, id);
            statement.setLong(2, USER_ID);
            statement.setString(3, word);
            statement.setString(4, word);
            statement.executeUpdate();
        }
    }

    private static void insertWord(Connection connection, String word, String sentence) throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement("""
                INSERT INTO word (id, word, sentence, status, difficulty, frequency)
                VALUES (1, ?, ?, 1, 1, 1)
                """)) {
            statement.setString(1, word);
            statement.setString(2, sentence);
            statement.executeUpdate();
        }
    }

    private static void insertBestSentence(Connection connection, long id, String word, String sentence)
            throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement("""
                INSERT INTO word_clean_best_sentence (id, word, sentence) VALUES (?, ?, ?)
                """)) {
            statement.setLong(1, id);
            statement.setString(2, word);
            statement.setString(3, sentence);
            statement.executeUpdate();
        }
    }

    private static void assertRows(
            Connection connection,
            String expectedSentence,
            String expectedSource,
            long expectedTotal,
            String expectedProgressKey
    ) throws SQLException {
        List<EventRow> rows = selectRows(connection);

        assertEquals(1, rows.size());
        assertEquals(expectedSentence, rows.getFirst().exampleSentence());
        assertEquals(expectedSource, rows.getFirst().exampleSource());
        assertEquals(expectedProgressKey, rows.getFirst().progressKey());
        assertEquals(rows.size(), rows.stream().map(EventRow::progressKey).distinct().count());
        assertEquals(expectedTotal, countRows(connection));
        if (expectedSentence == null) {
            assertNull(rows.getFirst().exampleSentence());
        }
    }

    private static List<EventRow> selectRows(Connection connection) throws SQLException {
        String sql = jdbcSql(WrongWordQueueEventSqlProvider.selectEvents());
        try (PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setLong(1, USER_ID);
            statement.setLong(2, USER_ID);
            statement.setLong(3, USER_ID);
            statement.setNull(4, java.sql.Types.VARCHAR);
            statement.setNull(5, java.sql.Types.VARCHAR);
            statement.setString(6, "recent");
            statement.setInt(7, 100);
            statement.setLong(8, 0L);
            try (ResultSet resultSet = statement.executeQuery()) {
                List<EventRow> rows = new ArrayList<>();
                while (resultSet.next()) {
                    rows.add(new EventRow(
                            resultSet.getString("progress_key"),
                            resultSet.getString("example_sentence"),
                            resultSet.getString("example_source")
                    ));
                }
                return rows;
            }
        }
    }

    private static long countRows(Connection connection) throws SQLException {
        String sql = jdbcSql(WrongWordQueueEventSqlProvider.countEvents());
        try (PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setLong(1, USER_ID);
            statement.setNull(2, java.sql.Types.VARCHAR);
            statement.setNull(3, java.sql.Types.VARCHAR);
            try (ResultSet resultSet = statement.executeQuery()) {
                resultSet.next();
                return resultSet.getLong(1);
            }
        }
    }

    private static String jdbcSql(String sql) {
        return sql.replaceAll("#\\{[a-zA-Z]+}", "?");
    }

    private static void execute(Connection connection, String sql) throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.execute();
        }
    }

    private record EventRow(String progressKey, String exampleSentence, String exampleSource) {
    }
}
