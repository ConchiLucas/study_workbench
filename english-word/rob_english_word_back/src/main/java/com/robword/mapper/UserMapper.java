package com.robword.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.robword.entity.User;
import org.apache.ibatis.annotations.Mapper;

@Mapper
public interface UserMapper extends BaseMapper<User> {
}