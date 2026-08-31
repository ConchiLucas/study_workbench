package com.robword.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.robword.entity.WordLibrary;
import org.apache.ibatis.annotations.Mapper;

@Mapper
public interface WordLibraryMapper extends BaseMapper<WordLibrary> {
}