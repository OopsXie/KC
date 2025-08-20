package org.example.minfsweb.result;
import lombok.Data;

import java.io.Serializable;
import java.util.UUID;

/**
 * 后端统一返回结果
 * @param <T>
 */
@Data
public class Result<T> implements Serializable {
    private static final String MSG_OK="ok";

    //编码：200成功，非200为失败
    private Integer code;
    //错误信息
    private String msg;
    //请求id，用于请求跟踪
    private String requestId;
    //数据
    private T data;

    public static <T> Result<T> success() {
        Result<T> result = new Result<T>();
        result.code = 200;
        result.msg = MSG_OK;
        result.requestId = UUID.randomUUID().toString();
        return result;
    }

    public static <T> Result<T> success(T object) {
        Result<T> result = new Result<T>();
        result.code = 200;
        result.msg = MSG_OK;
        result.data = object;
        result.requestId = UUID.randomUUID().toString();
        return result;
    }

    public static <T> Result<T> error(String msg) {
        Result<T> result = new Result<>();
        // 或其他非200的错误码
        result.code = 500;
        result.msg = msg;
        result.requestId = UUID.randomUUID().toString();
        return result;
    }

    public static <T> Result<T> error(Integer code, String msg) {
        Result<T> result = new Result<>();
        result.code = code;
        result.msg = msg;
        result.requestId = UUID.randomUUID().toString();
        return result;
    }

    public static Result failure(String message) {
        return Result.error(500, message);
    }
}