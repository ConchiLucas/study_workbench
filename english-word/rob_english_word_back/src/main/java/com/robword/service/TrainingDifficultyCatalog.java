package com.robword.service;

import org.springframework.stereotype.Component;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;

/**
 * 单人训练和正式匹配共用的难度目录。
 * group/level、展示名称和词库映射只能在这里定义，避免两个业务入口逐渐不一致。
 */
@Component
public class TrainingDifficultyCatalog {

    public record Difficulty(
            String group,
            String level,
            String label,
            List<String> libraryNames,
            boolean rankBased
    ) {
        public Difficulty {
            libraryNames = List.copyOf(libraryNames);
        }
    }

    private final Map<String, Difficulty> difficulties;

    public TrainingDifficultyCatalog() {
        Map<String, Difficulty> values = new LinkedHashMap<>();

        add(values, "rank", "rank_current", "段位难度", List.of(), true);

        List<String> primary = List.of(
                "PEPXiaoXue3_1", "PEPXiaoXue3_2", "PEPXiaoXue4_1", "PEPXiaoXue4_2",
                "PEPXiaoXue5_1", "PEPXiaoXue5_2", "PEPXiaoXue6_1", "PEPXiaoXue6_2"
        );
        add(values, "primary", "primary", "小学英语", primary, false);
        add(values, "primary", "primary_3_1", "小学英语 · 3年级上册", List.of("PEPXiaoXue3_1"), false);
        add(values, "primary", "primary_3_2", "小学英语 · 3年级下册", List.of("PEPXiaoXue3_2"), false);
        add(values, "primary", "primary_4_1", "小学英语 · 4年级上册", List.of("PEPXiaoXue4_1"), false);
        add(values, "primary", "primary_4_2", "小学英语 · 4年级下册", List.of("PEPXiaoXue4_2"), false);
        add(values, "primary", "primary_5_1", "小学英语 · 5年级上册", List.of("PEPXiaoXue5_1"), false);
        add(values, "primary", "primary_5_2", "小学英语 · 5年级下册", List.of("PEPXiaoXue5_2"), false);
        add(values, "primary", "primary_6_1", "小学英语 · 6年级上册", List.of("PEPXiaoXue6_1"), false);
        add(values, "primary", "primary_6_2", "小学英语 · 6年级下册", List.of("PEPXiaoXue6_2"), false);

        List<String> junior = List.of(
                "PEPChuZhong7_1", "PEPChuZhong7_2", "PEPChuZhong8_1", "PEPChuZhong8_2", "PEPChuZhong9_1"
        );
        add(values, "junior", "junior", "初中英语", junior, false);
        add(values, "junior", "junior_7_1", "初中英语 · 7年级上册", List.of("PEPChuZhong7_1"), false);
        add(values, "junior", "junior_7_2", "初中英语 · 7年级下册", List.of("PEPChuZhong7_2"), false);
        add(values, "junior", "junior_8_1", "初中英语 · 8年级上册", List.of("PEPChuZhong8_1"), false);
        add(values, "junior", "junior_8_2", "初中英语 · 8年级下册", List.of("PEPChuZhong8_2"), false);
        add(values, "junior", "junior_9_1", "初中英语 · 9年级上册", List.of("PEPChuZhong9_1"), false);

        List<String> senior = List.of(
                "PEPGaoZhong_1", "PEPGaoZhong_2", "PEPGaoZhong_3", "PEPGaoZhong_4", "PEPGaoZhong_5",
                "PEPGaoZhong_6", "PEPGaoZhong_7", "PEPGaoZhong_8", "PEPGaoZhong_9", "PEPGaoZhong_10", "PEPGaoZhong_11"
        );
        add(values, "senior", "senior", "高中英语", senior, false);
        for (int i = 1; i <= 11; i++) {
            String childLabel = i == 1 ? "上册" : i == 2 ? "下册" : "第" + i + "册";
            add(values, "senior", "senior_" + i, "高中英语 · " + childLabel,
                    List.of("PEPGaoZhong_" + i), false);
        }

        List<String> cet4 = List.of("CET4_1", "CET4_2", "CET4_3", "CET4luan_1", "CET4luan_2");
        List<String> cet6 = List.of("CET6_1", "CET6_2", "CET6_3", "CET6luan_1");
        add(values, "college", "college", "大学英语", concat(cet4, cet6), false);
        add(values, "college", "college_cet4", "大学英语 · 四级", cet4, false);
        add(values, "college", "college_cet6", "大学英语 · 六级", cet6, false);

        List<String> kaoyan = List.of("KaoYan_1", "KaoYan_2", "KaoYan_3", "KaoYanluan_1");
        add(values, "entrance", "entrance", "升学考试英语", kaoyan, false);
        add(values, "entrance", "entrance_kaoyan", "升学考试英语 · 考研", kaoyan, false);

        List<String> bec = List.of("BEC_2", "BEC_3");
        List<String> ielts = List.of("IELTS_2", "IELTS_3", "IELTSluan_2");
        List<String> toefl = List.of("TOEFL_2", "TOEFL_3");
        List<String> gmat = List.of("GMAT_2", "GMAT_3", "GMATluan_2");
        add(values, "business_abroad", "business_abroad", "商务与出国英语",
                concat(bec, ielts, toefl, gmat), false);
        add(values, "business_abroad", "business_bec", "商务与出国英语 · BEC", bec, false);
        add(values, "business_abroad", "business_ielts", "商务与出国英语 · 雅思", ielts, false);
        add(values, "business_abroad", "business_toefl", "商务与出国英语 · 托福", toefl, false);
        add(values, "business_abroad", "business_gmat", "商务与出国英语 · GMAT", gmat, false);

        List<String> tem4 = List.of("Level4_1", "Level4_2", "Level4luan_1", "Level4luan_2");
        List<String> tem8 = List.of("Level8_1", "Level8_2", "Level8luan_2");
        add(values, "professional", "professional", "专业英语", concat(tem4, tem8), false);
        add(values, "professional", "professional_tem4", "专业英语 · 专四级", tem4, false);
        add(values, "professional", "professional_tem8", "专业英语 · 专八级", tem8, false);

        List<String> gre = List.of("GRE_2", "GRE_3");
        List<String> sat = List.of("SAT_2", "SAT_3");
        add(values, "advanced_exam", "advanced_exam", "高阶考试英语", concat(gre, sat), false);
        add(values, "advanced_exam", "advanced_gre", "高阶考试英语 · GRE", gre, false);
        add(values, "advanced_exam", "advanced_sat", "高阶考试英语 · SAT", sat, false);

        difficulties = Map.copyOf(values);
    }

    public Optional<Difficulty> resolve(String group, String level) {
        if (group == null || group.isBlank() || level == null || level.isBlank()) {
            return Optional.empty();
        }
        Difficulty difficulty = difficulties.get(level);
        if (difficulty == null || !difficulty.group().equals(group)) {
            return Optional.empty();
        }
        return Optional.of(difficulty);
    }

    public Optional<Difficulty> resolveLevel(String level) {
        if (level == null || level.isBlank()) {
            return Optional.empty();
        }
        return Optional.ofNullable(difficulties.get(level));
    }

    private void add(Map<String, Difficulty> values,
                     String group,
                     String level,
                     String label,
                     List<String> libraries,
                     boolean rankBased) {
        values.put(level, new Difficulty(group, level, label, libraries, rankBased));
    }

    @SafeVarargs
    private static List<String> concat(List<String>... groups) {
        return java.util.Arrays.stream(groups).flatMap(List::stream).toList();
    }
}
