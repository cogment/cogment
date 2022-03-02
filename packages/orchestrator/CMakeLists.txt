add_subdirectory(dependencies)
add_subdirectory(third_party/slt_concur EXCLUDE_FROM_ALL)
add_subdirectory(third_party/slt_setting EXCLUDE_FROM_ALL)

find_program(CLANG_TIDY "clang-tidy")

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

add_subdirectory(lib)
add_subdirectory(dylib)


############################ code format ############################
if(NOT DEFINED CLANG_FORMAT_BIN)
  find_program(CLANG_FORMAT_BIN NAMES clang-format)
endif()

file(GLOB_RECURSE ALL_SOURCE_FILES *.cpp *.h *.cxx *.hxx *.hpp *.cc *.ipp)

set(CLANG_FORMAT_EXCLUDE_PATTERNS ${CLANG_FORMAT_EXCLUDE_PATTERNS}
  "/CMakeFiles/"
  "cmake"
  "third_party"
  "install"
  "generated"
)

foreach (SOURCE_FILE ${ALL_SOURCE_FILES})
    foreach (EXCLUDE_PATTERN ${CLANG_FORMAT_EXCLUDE_PATTERNS})
        string(FIND ${SOURCE_FILE} ${EXCLUDE_PATTERN} EXCLUDE_FOUND)
        if (NOT ${EXCLUDE_FOUND} EQUAL -1)
            list(REMOVE_ITEM ALL_SOURCE_FILES ${SOURCE_FILE})
        endif ()
    endforeach ()
endforeach ()


add_custom_target(orchestrator_lint_fix
  COMMAND ${CLANG_FORMAT_BIN} -style=file -i ${ALL_SOURCE_FILES}
  COMMENT "Running clang-format to fix the format of 'orchestrator' files"
  VERBATIM
)

add_custom_target(orchestrator_lint
  COMMAND ${CLANG_FORMAT_BIN} -style=file --dry-run -Werror ${ALL_SOURCE_FILES}
  COMMENT "Running clang-format to fix the format of 'orchestrator' files"
  VERBATIM
)
#############################################################
