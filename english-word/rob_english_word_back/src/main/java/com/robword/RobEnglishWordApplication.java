package com.robword;

import org.mybatis.spring.annotation.MapperScan;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication
@EnableScheduling
@MapperScan("com.robword.mapper")
public class RobEnglishWordApplication {

    public static void main(String[] args) {
        SpringApplication.run(RobEnglishWordApplication.class, args);
    }
}