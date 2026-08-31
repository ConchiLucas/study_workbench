package com.robword.mapper;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.condition.EnabledIfEnvironmentVariable;
import org.apache.ibatis.annotations.Insert;

import java.nio.file.Files;
import java.nio.file.Path;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Savepoint;
import java.sql.Timestamp;
import java.lang.reflect.Method;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

@EnabledIfEnvironmentVariable(named = "RUN_POSTGRES_INTEGRATION", matches = "true")
class SentenceClozeMigrationPostgresTest {

    private static final long USER_ID = 77L;

    @Test
    void migrationIsRepeatableAndBackfillsOnlyUnfinishedWrongHistory() throws Exception {
        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                execute(connection, "SET LOCAL search_path TO pg_temp");
                createLegacyFixture(connection);
                insertFixtureHistory(connection);

                String answerDdl = tempSchemaSql("db/sentence_cloze_answer_record.sql");
                String scheduleDdl = tempSchemaSql("db/sentence_cloze_review_schedule.sql");
                execute(connection, answerDdl);
                execute(connection, scheduleDdl);
                Timestamp firstMigrationUpdateTime = scalarTimestamp(connection, """
                        SELECT update_time FROM sentence_cloze_review_schedule
                        WHERE user_id = 77 AND cloze_item_id = 10
                        """);
                execute(connection, answerDdl);
                execute(connection, scheduleDdl);

                assertEquals(2, scalarLong(connection, """
                        SELECT COUNT(*) FROM sentence_cloze_review_schedule WHERE user_id = 77
                        """));
                assertEquals(2, scalarLong(connection, """
                        SELECT review_stage FROM sentence_cloze_review_schedule
                        WHERE user_id = 77 AND cloze_item_id = 10
                        """));
                assertEquals(1, scalarLong(connection, """
                        SELECT COUNT(*) FROM sentence_cloze_review_schedule
                        WHERE user_id = 77 AND cloze_item_id = 20
                          AND status = 'active' AND review_stage = 0 AND wrong_count = 1
                        """));
                assertEquals(0, scalarLong(connection, """
                        SELECT COUNT(*) FROM sentence_cloze_review_schedule
                        WHERE user_id = 77 AND cloze_item_id = 30
                        """));
                assertEquals(0, scalarLong(connection, """
                        SELECT COUNT(*) FROM sentence_cloze_review_schedule
                        WHERE user_id = 77 AND cloze_item_id = 40
                        """));
                assertEquals(100, scalarLong(connection, """
                        SELECT last_wrong_answer_record_id FROM sentence_cloze_review_schedule
                        WHERE user_id = 77 AND cloze_item_id = 10
                        """));
                assertEquals(firstMigrationUpdateTime, scalarTimestamp(connection, """
                        SELECT update_time FROM sentence_cloze_review_schedule
                        WHERE user_id = 77 AND cloze_item_id = 10
                        """));

                execute(connection, """
                        INSERT INTO sentence_cloze_answer_record (
                            id, user_id, cloze_item_id, submission_key
                        ) VALUES (500, 77, 20, 'duplicate-key')
                        """);
                Savepoint beforeDuplicate = connection.setSavepoint();
                assertThrows(SQLException.class, () -> execute(connection, """
                        INSERT INTO sentence_cloze_answer_record (
                            id, user_id, cloze_item_id, submission_key
                        ) VALUES (501, 77, 20, 'duplicate-key')
                        """));
                connection.rollback(beforeDuplicate);
            } finally {
                connection.rollback();
            }
        }
    }

    @Test
    void targetWordSearchMatchesEveryBlankInPostgres() throws Exception {
        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                execute(connection, "SET LOCAL search_path TO pg_temp");
                execute(connection, """
                        CREATE TEMP TABLE sentence_cloze_item (
                            id bigint PRIMARY KEY,
                            word text,
                            blank_words_json text NOT NULL
                        )
                        """);
                execute(connection, """
                        INSERT INTO sentence_cloze_item (id, word, blank_words_json)
                        VALUES (1, 'redintegrate', '["redintegrate", "sacrilegious", "fracture"]')
                        """);

                assertEquals(1, scalarLong(connection, """
                        SELECT COUNT(*)
                        FROM sentence_cloze_item item
                        WHERE EXISTS (
                            SELECT 1
                            FROM jsonb_array_elements_text(item.blank_words_json::jsonb) target_word(word)
                            WHERE LOWER(target_word.word) LIKE '%fracture%'
                        )
                        """));
            } finally {
                connection.rollback();
            }
        }
    }

    @Test
    void wrongScheduleUpsertRejectsStaleAndReplayedAnswerRecords() throws Exception {
        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                execute(connection, "SET LOCAL search_path TO pg_temp");
                createLegacyFixture(connection);
                execute(connection, """
                        INSERT INTO sentence_cloze_answer_record
                            (id, user_id, cloze_item_id, is_correct, create_time)
                        VALUES
                            (100, 77, 20, false, '2026-07-02 10:00:00'),
                            (101, 77, 20, true,  '2026-07-02 10:01:00')
                        """);
                execute(connection, tempSchemaSql("db/sentence_cloze_answer_record.sql"));
                execute(connection, tempSchemaSql("db/sentence_cloze_review_schedule.sql"));

                String staleUpsert = wrongScheduleUpsertSql(77, 20, 100);
                execute(connection, staleUpsert);
                assertEquals(0, scalarLong(connection, """
                        SELECT COUNT(*) FROM sentence_cloze_review_schedule
                        WHERE user_id = 77 AND cloze_item_id = 20
                        """));

                execute(connection, """
                        INSERT INTO sentence_cloze_answer_record
                            (id, user_id, cloze_item_id, is_correct, create_time)
                        VALUES (102, 77, 20, false, '2026-07-02 10:02:00')
                        """);
                String currentUpsert = wrongScheduleUpsertSql(77, 20, 102);
                execute(connection, currentUpsert);
                execute(connection, currentUpsert);
                assertEquals(1, scalarLong(connection, """
                        SELECT wrong_count FROM sentence_cloze_review_schedule
                        WHERE user_id = 77 AND cloze_item_id = 20
                        """));
                assertEquals(102, scalarLong(connection, """
                        SELECT last_answer_record_id FROM sentence_cloze_review_schedule
                        WHERE user_id = 77 AND cloze_item_id = 20
                        """));
            } finally {
                connection.rollback();
            }
        }
    }

    private static String wrongScheduleUpsertSql(long userId, long clozeItemId, long recordId) throws Exception {
        Method method = SentenceClozeReviewScheduleMapper.class.getMethod(
                "upsertWrongSchedule", Long.class, Long.class, Long.class, java.time.LocalDateTime.class
        );
        return String.join("\n", method.getAnnotation(Insert.class).value())
                .replace("#{userId}", Long.toString(userId))
                .replace("#{clozeItemId}", Long.toString(clozeItemId))
                .replace("#{recordId}", Long.toString(recordId));
    }

    private static Connection openConnection() throws SQLException {
        return DriverManager.getConnection(
                requiredEnvironment("SPRING_DATASOURCE_URL"),
                requiredEnvironment("SPRING_DATASOURCE_USERNAME"),
                requiredEnvironment("SPRING_DATASOURCE_PASSWORD")
        );
    }

    private static String requiredEnvironment(String name) {
        String value = System.getenv(name);
        if (value == null || value.isBlank()) {
            throw new IllegalStateException(name + " must be set when RUN_POSTGRES_INTEGRATION=true");
        }
        return value;
    }

    private static String tempSchemaSql(String path) throws Exception {
        return Files.readString(Path.of(path)).replace("public.", "pg_temp.");
    }

    private static void createLegacyFixture(Connection connection) throws SQLException {
        execute(connection, "CREATE TEMP SEQUENCE sentence_cloze_review_schedule_id_seq");
        execute(connection, """
                CREATE TEMP TABLE sentence_cloze_answer_record (
                    id bigint PRIMARY KEY,
                    user_id bigint NOT NULL,
                    user_name varchar(100),
                    cloze_item_id bigint NOT NULL,
                    answer_text text NOT NULL DEFAULT '',
                    answers_json text NOT NULL DEFAULT '[]',
                    expected_words_json text NOT NULL DEFAULT '[]',
                    is_correct boolean NOT NULL DEFAULT false,
                    attempt_no int NOT NULL DEFAULT 1,
                    cost_ms bigint,
                    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
                    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
                )
                """);
        execute(connection, """
                CREATE TEMP TABLE sentence_cloze_review_schedule (
                    id bigint PRIMARY KEY DEFAULT nextval('sentence_cloze_review_schedule_id_seq'),
                    user_id bigint NOT NULL,
                    cloze_item_id bigint NOT NULL,
                    correct_streak int NOT NULL DEFAULT 0,
                    review_stage int NOT NULL DEFAULT 0,
                    next_review_time timestamp NOT NULL,
                    last_answer_record_id bigint,
                    last_wrong_time timestamp,
                    last_correct_time timestamp,
                    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
                    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
                )
                """);
    }

    private static void insertFixtureHistory(Connection connection) throws SQLException {
        execute(connection, """
                INSERT INTO sentence_cloze_answer_record
                    (id, user_id, cloze_item_id, is_correct, create_time)
                VALUES
                    (100, 77, 10, false, '2026-07-01 10:00:00'),
                    (200, 77, 20, false, '2026-07-02 10:00:00'),
                    (300, 77, 30, false, '2026-07-03 10:00:00'),
                    (301, 77, 30, true,  '2026-07-04 10:00:00'),
                    (302, 77, 30, true,  '2026-07-12 10:00:00'),
                    (303, 77, 30, true,  '2026-07-28 10:00:00'),
                    (400, 77, 40, false, '2026-07-05 10:00:00'),
                    (401, 77, 40, true,  '2026-07-06 10:00:00')
                """);
        execute(connection, """
                INSERT INTO sentence_cloze_review_schedule (
                    id, user_id, cloze_item_id, correct_streak, review_stage,
                    next_review_time, last_answer_record_id, last_wrong_time
                ) VALUES (
                    1, 77, 10, 2, 2, '2026-08-20 10:00:00', 100, '2026-07-01 10:00:00'
                )
                """);
    }

    private static long scalarLong(Connection connection, String sql) throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement(sql);
             ResultSet resultSet = statement.executeQuery()) {
            resultSet.next();
            return resultSet.getLong(1);
        }
    }

    private static Timestamp scalarTimestamp(Connection connection, String sql) throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement(sql);
             ResultSet resultSet = statement.executeQuery()) {
            resultSet.next();
            return resultSet.getTimestamp(1);
        }
    }

    private static void execute(Connection connection, String sql) throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.execute();
        }
    }
}
