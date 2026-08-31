package com.robword.service;

import com.robword.entity.Word;
import com.robword.mapper.WordMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.concurrent.TimeUnit;

@Service
@RequiredArgsConstructor
public class WordService {

    private final WordMapper wordMapper;
    private final RedisTemplate<String, Object> redisTemplate;

    private static final String WORD_CACHE_KEY = "word:cache:";
    private static final String ALL_WORDS_KEY = "word:all:enabled";

    public List<Word> getRandomWords(int minDifficulty, int maxDifficulty, int count) {
        String cacheKey = WORD_CACHE_KEY + minDifficulty + ":" + maxDifficulty + ":" + count;

        @SuppressWarnings("unchecked")
        List<Word> cached = (List<Word>) redisTemplate.opsForValue().get(cacheKey);
        if (cached != null) {
            return cached;
        }

        List<Word> words = wordMapper.findRandomWords(minDifficulty, maxDifficulty, count);
        redisTemplate.opsForValue().set(cacheKey, words, 30, TimeUnit.MINUTES);
        return words;
    }

    /**
     * 获取随机单词（无缓存），用于匹配游戏，每次调用都返回不同的单词
     */
    public List<Word> getRandomWordsForMatch(int minDifficulty, int maxDifficulty, int count) {
        // 直接查询数据库，不使用缓存，确保每次匹配都是不同的单词
        return wordMapper.findRandomWords(minDifficulty, maxDifficulty, count);
    }

    public List<Word> getRandomWordsForSoloTraining(Long userId,
                                                    int minDifficulty,
                                                    int maxDifficulty,
                                                    int count,
                                                    List<Long> excludeWordIds) {
        if (count <= 0) {
            return List.of();
        }
        return wordMapper.findRandomWordsExcludingMastered(userId, minDifficulty, maxDifficulty, count, excludeWordIds);
    }

    public List<Word> getRandomWordsForTrainingLibraries(List<String> libraryNames, int count) {
        if (libraryNames == null || libraryNames.isEmpty()) {
            return List.of();
        }
        return wordMapper.findRandomWordsByLibraryNames(libraryNames, count);
    }

    public List<Word> getRandomWordsForTrainingLibraries(Long userId,
                                                         List<String> libraryNames,
                                                         int count,
                                                         List<Long> excludeWordIds) {
        if (libraryNames == null || libraryNames.isEmpty() || count <= 0) {
            return List.of();
        }
        return wordMapper.findRandomWordsByLibraryNamesExcludingMastered(userId, libraryNames, count, excludeWordIds);
    }

    public void clearWordCache() {
        redisTemplate.delete(redisTemplate.keys(WORD_CACHE_KEY + "*"));
        redisTemplate.delete(ALL_WORDS_KEY);
    }
}
